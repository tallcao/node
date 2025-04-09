package model

type Converter interface {
	SendFrame([]byte)
	GetSN() string
}

type AddrAdapter interface {
	GetAddr() uint8
}

// type WifiDevice interface {
// 	Get() ([]byte, error)

// 	// SetGUID(string)
// 	GetGUID() string

// 	// SetConnected(bool)
// 	GetConnected() bool

// 	GetFullSN() string

// 	ReceiveFunc(string, string, []byte) error
// 	CommandFunc(string, interface{}) (string, []byte, error)
// }
