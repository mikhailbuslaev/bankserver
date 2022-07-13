package server

import (
	"fmt"
	"log"
	"os"
	"sync"
	"math"
	"os/signal"
	"syscall"
	"time"
	"bankservice/cache"
	"bankservice/user"
	"strconv"
	"github.com/valyala/fasthttp"
	"golang.org/x/crypto/bcrypt"
	"encoding/json"
	"gopkg.in/yaml.v3"
)

type BankService struct {
	Server *fasthttp.Server
	Cache *cache.Cache
	WaitGroup *sync.WaitGroup
	Config *BankServiceConfig
}

type BankServiceConfig struct {
	StorageName string `yaml:"storagename"`
	Port string `yaml:"port"`
}

type OkResponse struct {
	Result string `json:"result"`
}

type ErrResponse struct {
	Result string `json:"error"`
}

func WriteBody(ctx *fasthttp.RequestCtx, v interface{}) {
	resp, err := json.Marshal(v)
	if err != nil {
		ctx.SetStatusCode(500)
		ctx.Write([]byte(`{"error": "server cannot make response"}`))
		return
	}
	ctx.Write(resp)
}

func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
    return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}

func (b *BankService) GetBalanceHandler(ctx *fasthttp.RequestCtx) {
	id := string(ctx.Request.Header.Peek("id"))
	password := string(ctx.Request.Header.Peek("password"))

	user, err := b.Cache.Get(id)
	if err != nil {
		ctx.SetStatusCode(400)
		WriteBody(ctx, ErrResponse{"user not exists"})
		return
	}
	if !CheckPasswordHash(password, user.AuthKey) {
		ctx.SetStatusCode(400)
		WriteBody(ctx, ErrResponse{"wrong password"})
		return
	}
	ctx.SetStatusCode(200)
	WriteBody(ctx, OkResponse{strconv.FormatFloat(user.Balance, 'f', 2, 64)})
}

func (b *BankService) MakeTransactionHandler(ctx *fasthttp.RequestCtx) {
	senderId := string(ctx.Request.Header.Peek("sender_id"))
	receiverId := string(ctx.Request.Header.Peek("receiver_id"))
	password := string(ctx.Request.Header.Peek("password"))

	sum, err := strconv.ParseFloat(string(ctx.Request.Header.Peek("sum")), 64)
	if err != nil || sum < 0.00 || math.Round(sum*100)/100 != sum {
		ctx.SetStatusCode(400)
		WriteBody(ctx, ErrResponse{"invalid sum"})
		return
	}

	sender, err := b.Cache.Get(senderId)
	if err != nil {
		ctx.SetStatusCode(400)
		WriteBody(ctx, ErrResponse{"sender not exists"})
		return
	}

	_, err = b.Cache.Get(receiverId)
	if err != nil {
		ctx.SetStatusCode(400)
		WriteBody(ctx, ErrResponse{"receiver not exists"})
		return
	}
	if !CheckPasswordHash(password, sender.AuthKey) {
		ctx.SetStatusCode(400)
		WriteBody(ctx, ErrResponse{"wrong password"})
		return
	}

	if err := b.Cache.ChangeBalance(senderId, -sum); err != nil {
		ctx.SetStatusCode(500)
		WriteBody(ctx, ErrResponse{"transaction failed: "+err.Error()})
		return
	}
	if err := b.Cache.ChangeBalance(receiverId, sum); err != nil {
		ctx.SetStatusCode(500)
		WriteBody(ctx, ErrResponse{"transaction failed: "+err.Error()})
		return
	}
	ctx.SetStatusCode(200)
	WriteBody(ctx, OkResponse{"successful transaction"})
}

func (b *BankService) CreateUserHandler(ctx *fasthttp.RequestCtx) {
	id := string(ctx.Request.Header.Peek("id"))
	hashPass, err := HashPassword(string(ctx.Request.Header.Peek("password")))
	if err != nil {
		ctx.SetStatusCode(400)
		WriteBody(ctx, ErrResponse{"invalid password"})
		return
	}
	balance, err := strconv.ParseFloat(string(ctx.Request.Header.Peek("balance")), 64)
	if err != nil || balance < 0.00 || math.Round(balance*100)/100 != balance {
		ctx.SetStatusCode(400)
		WriteBody(ctx, ErrResponse{"invalid balance"})
		return
	}
	user := user.Create(id, hashPass, balance)
	if err := b.Cache.Add(user); err != nil {
		ctx.SetStatusCode(500)
		WriteBody(ctx, ErrResponse{"user creating failed"})
		return
	}
	ctx.SetStatusCode(200)
	WriteBody(ctx, OkResponse{"successful user creating"})
}

func (b *BankService) LoadConfig(configName string) {
	buf, err := os.ReadFile(configName)
	if err != nil {
		log.Fatalf("Cannot read config file")
	}

	err = yaml.Unmarshal(buf, &b.Config)
	if err != nil {
		log.Println(err)
		log.Fatalf("Cannot parse config")
	}
}

func New() *BankService {
	b := &BankService{}
	b.Cache = cache.New()
	b.WaitGroup = &sync.WaitGroup{}
	b.LoadConfig("config.yaml")

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		b.WaitGroup.Add(1)
		switch string(ctx.Path()) {
		case "/get_balance":
			b.GetBalanceHandler(ctx)
		case "/transaction":
			b.MakeTransactionHandler(ctx)
		case "/create_user":
			b.CreateUserHandler(ctx)
		}
		b.WaitGroup.Done()
	}

	b.Server = &fasthttp.Server{
		Handler: requestHandler,
		Name: "My bank server",
	}
	return b
}

func (b *BankService) Run() {
	fmt.Println("server run...")
	// last cache backup
	defer func() {
		os.Truncate(b.Config.StorageName, 0)
		b.Cache.ScreenToFile(b.Config.StorageName)
	}()
	// get cache from file
	if err := b.Cache.RestoreFromFile(b.Config.StorageName); err != nil {
		log.Fatalf("restore cash fail")
	}
	// fasthttp server run
	go func() {
		if err := b.Server.ListenAndServe(b.Config.Port); err != nil {
			log.Fatalf("error in ListenAndServe: %v", err)
		}
	}()
	// backup cache every 5 min, we need this because SIGKILL can happen
	go func() {
		for {
			b.WaitGroup.Add(1)
			os.Truncate(b.Config.StorageName, 0)
			b.Cache.ScreenToFile(b.Config.StorageName)
			b.WaitGroup.Done()
			time.Sleep(5*time.Minute)
		}
	}()
	// wait SIGINT
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
	<- exit
	println("server exit...")
	// wait until all workers done
	b.WaitGroup.Wait()
	time.Sleep(1*time.Second)
}
