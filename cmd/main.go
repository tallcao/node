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
	// flag.StringVar(&config, "c", "zigbee.json", "config file")
	flag.StringVar(&config, "c", "node.json", "config file")
}

func main() {

	dbusService := service.NewDbusService()

	go dbusService.Run()

	node := core.NewNode(config, dbusService)

	node.Init()
	node.Run()
}
