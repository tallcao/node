package model

type Connection struct {
	Rx chan []byte
	Tx chan []byte
}
