package main

import "bankservice/server"

func main() {
	bankserver := server.New()
	bankserver.Run()
}