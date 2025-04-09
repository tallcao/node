package main

import (
	"edge/core"
	"edge/service"
)

func main() {

	dbService := service.NewDbusService()

	go dbService.Run()

	file := "/home/root/node/node.json"
	node := core.NewNode(file, dbService)

	node.Init()
	node.Run()
}
