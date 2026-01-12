package service

import (
	"edge/model"
	"fmt"
	"net"
	"os"
)

const SerialSocketPath = "/tmp/serial_socket"
const LoraSocketPath = "/tmp/lora_socket"
const CanSocketPath = "/tmp/can_socket"

type ConnectionChan struct {
	socketPath string
	clientConn net.PacketConn
	addr       net.Addr

	connection model.Connection
}

type UdsServer struct {
	serail ConnectionChan
	lora   ConnectionChan
	can    ConnectionChan
}

func startUdsServer(ch *ConnectionChan) {
	// 1. 启动前清理：如果 Socket 文件已存在，必须先删除，否则会报 "address already in use"
	if err := os.RemoveAll(ch.socketPath); err != nil {
		fmt.Printf("Failed to remove socket path : %v\n", err)
		return
	}

	conn, _ := net.ListenPacket("unixgram", ch.socketPath)
	ch.clientConn = conn
	defer conn.Close()

	os.Chmod(ch.socketPath, 0777)

	fmt.Printf("UDS server start, socket path: %s\n", ch.socketPath)

	buf := make([]byte, 256)
	for {
		n, addr, _ := conn.ReadFrom(buf)

		if n > 0 {

			if n == 1 {
				ch.addr = addr
				continue
			}
			ch.addr = addr
			tmp := make([]byte, n)
			copy(tmp, buf[:n])

			ch.connection.Rx <- tmp
		}
	}

}

func NewUdsServer() *UdsServer {
	serail := ConnectionChan{
		socketPath: SerialSocketPath,
		connection: model.Connection{
			Rx: make(chan []byte, 128),
			Tx: make(chan []byte, 128),
		},
	}

	lora := ConnectionChan{
		socketPath: LoraSocketPath,
		connection: model.Connection{
			Rx: make(chan []byte, 128),
			Tx: make(chan []byte, 128),
		},
	}

	can := ConnectionChan{
		socketPath: CanSocketPath,
		connection: model.Connection{
			Rx: make(chan []byte, 128),
			Tx: make(chan []byte, 128),
		},
	}

	s := &UdsServer{
		serail: serail,
		lora:   lora,
		can:    can,
	}

	return s
}

func (s *UdsServer) CanConnection() model.Connection {

	return s.can.connection
}

func (s *UdsServer) LoraConnection() model.Connection {

	return s.lora.connection
}
func (s *UdsServer) SerialConnection() model.Connection {
	return s.serail.connection
}

func (s *UdsServer) Run() {

	go startUdsServer(&s.serail)
	go startUdsServer(&s.lora)
	go startUdsServer(&s.can)

	// tx

	go handleTx(&s.can)
	go handleTx(&s.serail)
	go handleTx(&s.lora)

	select {}
}

func handleTx(ch *ConnectionChan) {
	for data := range ch.connection.Tx {

		if ch.clientConn != nil && ch.addr != nil {
			ch.clientConn.WriteTo(data, ch.addr)
		}
	}
}
