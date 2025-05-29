package model

type Device485 interface {
	GetDevice485Setting() (uint32, byte, byte, byte)
}

type DeviceRelay interface {
	GetRelayDefaultState() byte
}

type Converter interface {
	SendFrame([]byte)
	GetSN() string
	HeartRequest()
}

type AddrAdapter interface {
	GetAddr() uint8
}

type Parent interface {
	GetChildren() []SparkplugDevice
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

type IHeart interface {
	HeartBeat()
	HeartCheck()
	IsConnected() bool
	ConnectedChanged() bool
}

type Thing interface {
	Response([]byte)
	Request(string, interface{})
	GetId() string
	Converter
	// ReportOnConnected()

	GetType() DEVICE_TYPE

	IHeart

	SparkplugDevice
}
