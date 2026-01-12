package main

import (
	"edge/core"
	"edge/service"
	"flag"
)

var (
	config string
)

func init() {
	flag.StringVar(&config, "c", "node.json", "config file")
}

func main() {

	udsServer := service.NewUdsServer()
	go udsServer.Run()

	node := core.NewNode(config, udsServer)

	node.Run()
}
