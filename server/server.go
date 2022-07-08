package server

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"bankservice/cache"
	"bankservice/user"
	"strconv"
	"github.com/valyala/fasthttp"
	"golang.org/x/crypto/bcrypt"
	"encoding/json"
)

type BankService struct {
	Server *fasthttp.Server
	Cache *cache.Cache
	DoneChan chan struct{}
}

type OkResponse struct {
	Result interface{} `json:"result"`
}

type ErrResponse struct {
	Result interface{} `json:"error"`
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
	WriteBody(ctx, OkResponse{user.Balance})
}

func (b *BankService) MakeTransactionHandler(ctx *fasthttp.RequestCtx) {
	senderId := string(ctx.Request.Header.Peek("sender_id"))
	receiverId := string(ctx.Request.Header.Peek("receiver_id"))
	password := string(ctx.Request.Header.Peek("password"))

	sum, err := strconv.ParseFloat(string(ctx.Request.Header.Peek("sum")), 64)
	if err != nil || sum < 0.00 {
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
		WriteBody(ctx, ErrResponse{"transaction failed"})
		return
	}
	if err := b.Cache.ChangeBalance(receiverId, sum); err != nil {
		ctx.SetStatusCode(500)
		WriteBody(ctx, ErrResponse{"transaction failed"})
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
	if err != nil || balance < 0.00 {
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
	if err = b.Cache.PushToFile("cache.csv", user); err != nil {
		ctx.SetStatusCode(500)
		WriteBody(ctx, ErrResponse{"saving to file fail"})
		return
	}
	ctx.SetStatusCode(200)
	WriteBody(ctx, OkResponse{"successful user creating"})
}

func New() *BankService {
	b := &BankService{}
	b.Cache = cache.New()
	b.DoneChan = make(chan struct{}, 1)
	requestHandler := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/get_balance":
			b.GetBalanceHandler(ctx)
		case "/transaction":
			b.MakeTransactionHandler(ctx)
		case "/create_user":
			b.CreateUserHandler(ctx)
		}
		b.DoneChan <- struct{}{}
	}

	b.Server = &fasthttp.Server{
		Handler: requestHandler,
		Name: "My bank server",
	}
	return b
}

func (b *BankService) Run() {
	if err := b.Cache.RestoreFromFile("cache.csv"); err != nil {
		log.Fatalf("restore cash fail")
	}
	fmt.Println("server run...")
	go func() {
		if err := b.Server.ListenAndServe(":1111"); err != nil {
			log.Fatalf("error in ListenAndServe: %v", err)
		}
	}()

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
	<- exit
	
	println("server exit...")
	<-b.DoneChan
	time.Sleep(3*time.Second)
}
