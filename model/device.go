package model

const (
	ConnectionTypeCAN    = "can"
	ConnectionTypeLora   = "lora"
	ConnectionTypeSerial = "serial"
)

type Device struct {
	UUID           string `json:"uuid"`
	Vendor         string `json:"vendor,omitempty"`
	Model          string `json:"model,omitempty"`
	ConverterSN    string `json:"converter_sn,omitempty"`
	ConverterType  int    `json:"converter_type,omitempty"`
	CanID          byte   `json:"can_id,omitempty"`
	Addr           byte   `json:"addr,omitempty"`
	Operation      string `json:"operation"`
	ConnectionType string `json:"connection_type"`

	Children []*DeviceChild `json:"children,omitempty"`
}

type DevicesUpdate struct {
	Devices []Device `json:"devices"`
	Version int64    `json:"version"`
}

type DeviceChild struct {
	UUID string `json:"uuid"`
	No   int    `json:"no"`
}
