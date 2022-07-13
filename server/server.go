package server

import (
	"fmt"
	"log"
	"os"
	"sync"
	"os/signal"
	"syscall"
	"time"
	"bankservice/cache"
	"github.com/valyala/fasthttp"
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
