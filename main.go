package main

import (
	"os/signal"
	"os"
	"syscall"
	"bankservice/server"
	"github.com/valyala/fasthttp"
	"strconv"
	"math/rand"
	"sync"
	"time"
)

func request() {
	rand.Seed(time.Now().UnixNano())
	sender_id := strconv.FormatInt(int64(rand.Intn(11)), 16)
	receiver_id := strconv.FormatInt(int64(rand.Intn(11)), 16)

	req := fasthttp.AcquireRequest()
    req.SetRequestURI("http://localhost:1111/transaction")
    req.Header.SetMethod("POST")
    req.SetBodyString("p=q")
	req.Header.Add("sender_id", string(sender_id))
	req.Header.Add("receiver_id", string(receiver_id))
	req.Header.Add("sum", "100.00")
	req.Header.Add("password", "12345")

    resp := fasthttp.AcquireResponse()
    client := &fasthttp.Client{}
    client.Do(req, resp)
	println("transaction done")
}

func main() {
	bankserver := server.New()
	wg := sync.WaitGroup{}
	go bankserver.Run()
	time.Sleep(5*time.Second)

	for i:=0;i<10;i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			request()
			wg.Done()
		}(&wg)
	}
	wg.Wait()

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
	<- exit
	time.Sleep(5*time.Second)
}
