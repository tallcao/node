package service

import (
	"edge/model"
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
)

type DbusService struct {
	connections map[string]model.Connection
}

func (s DbusService) ReadDataFrame(t string, data []byte) (bool, *dbus.Error) {
	if conn, found := s.connections[t]; found {
		go func() {
			conn.Rx <- data
		}()

	}

	return true, nil
}

func sendDataFrame(conn *dbus.Conn, data []byte, t string) error {
	dest := fmt.Sprintf("smart.device.%v", t)
	path := dbus.ObjectPath(fmt.Sprintf("/smart/device/%v", t))
	obj := conn.Object(dest, path)
	method := fmt.Sprintf("smart.device.%v.WriteData", t)
	err := obj.Call(method, 0, data).Err
	if err != nil {
		return err
	}
	return nil
}

func NewDbusService() *DbusService {

	canConn := model.Connection{
		Rx: make(chan []byte),
		Tx: make(chan []byte),
	}
	loraConn := model.Connection{
		Rx: make(chan []byte),
		Tx: make(chan []byte),
	}
	serialConn := model.Connection{
		Rx: make(chan []byte),
		Tx: make(chan []byte),
	}

	s := &DbusService{
		connections: make(map[string]model.Connection, 3),
	}

	s.connections["can"] = canConn
	s.connections["lora"] = loraConn
	s.connections["serial"] = serialConn

	return s
}

func export(conn *dbus.Conn, s *DbusService) {

	conn.Export(s, "/smart/gateway", "smart.gateway")

	reply, err := conn.RequestName("smart.gateway", dbus.NameFlagReplaceExisting)
	if err != nil {
		panic(err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		panic("request name already taken")
	}

}

func (s *DbusService) CanConnection() model.Connection {
	return s.connections["can"]
}
func (s *DbusService) LoraConnection() model.Connection {
	return s.connections["lora"]
}
func (s *DbusService) SerialConnection() model.Connection {
	return s.connections["serial"]
}

func (s *DbusService) Run() {

	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	export(conn, s)

	go dbusTx(s.connections["can"].Tx, conn, "can", 0)
	go dbusTx(s.connections["lora"].Tx, conn, "lora", 0)
	go dbusTx(s.connections["serial"].Tx, conn, "serial", time.Millisecond*300)

	select {}

}

func dbusTx(tx <-chan []byte, conn *dbus.Conn, t string, d time.Duration) {
	for data := range tx {
		sendDataFrame(conn, data, t)
		time.Sleep(d)
	}
}
