package model

import (
	"edge/utils"
	"encoding/json"
	"time"

	"google.golang.org/protobuf/proto"
)

type LightModule16 struct {
	status *Payload_Metric

	addr byte

	code int

	guid string

	observerList []Observer
	converter    Converter

	heart *Heart
}

type childData struct {
	Data int `json:"data"`
}

func NewLightModule16(guid string, c Converter, o Observer) *LightModule16 {

	item := &LightModule16{
		status: &Payload_Metric{
			Name:      proto.String("status"),
			Datatype:  proto.Uint32(uint32(DataType_Bytes)),
			Value:     &Payload_Metric_BytesValue{make([]byte, 16)},
			Timestamp: proto.Uint64(0),
		},

		guid: guid,

		converter: c,

		heart: new(Heart),
	}

	item.register(o)

	if adapter, ok := c.(AddrAdapter); ok {
		item.addr = adapter.GetAddr()
	} else {
		item.addr = 0x01
	}

	return item
}

func (d *LightModule16) Request(command string, params interface{}) {

	var data []byte

	switch command {
	case "heartBeat":
		data = []byte{d.addr, 0x03, 0x01, 0x00, 0x00, 0x02}
	case "getStatus":
		data = []byte{d.addr, 0x03, 0x01, 0x00, 0x00, 0x02}
	case "on", "off", "toggle":

		jsonData, err := json.Marshal(params)
		if err != nil {
			return
		}

		p := &childData{}
		err = json.Unmarshal(jsonData, &p)
		if p == nil || err != nil {
			return
		}

		i := p.Data

		if i < 1 || i > 16 {
			return
		}

		var cmd byte

		if command == "off" {
			cmd = 0x00
		}
		if command == "on" {
			cmd = 0x02
		}

		status := d.status.GetBytesValue()

		if command == "toggle" {

			cmd = 0x02
			if status[i-1] == 1 {
				cmd = 0x00
			}
		}

		if i < 9 {
			cmd += 2
		} else {
			cmd += 3
		}

		subAddr := 1 << ((16 - i) % 8)

		data = []byte{d.addr, 0x06, 0x01, cmd, 0x00, byte(subAddr)}

	case "fullOn":
		data = []byte{d.addr, 0x10, 0x01, 0x02, 0x00, 0x04, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0xff}
		d.code = 3
	case "fullOff":
		data = []byte{d.addr, 0x10, 0x01, 0x02, 0x00, 0x04, 0x08, 0x00, 0xff, 0x00, 0xff, 0x00, 0x00, 0x00, 0x00}
		d.code = 4

	default:
		return
	}

	crc, err := utils.CRC16(data)
	if err != nil {
		return
	}
	data = append(data, crc...)

	d.converter.SendFrame(data)

}

func (d *LightModule16) Response(data []byte) {

	if len(data) != 8 && len(data) != 9 {
		return
	}

	cmd := data[1]

	*d.status.Timestamp = uint64(time.Now().UnixMicro())

	status := d.status.GetBytesValue()

	// full open, full close
	if cmd == 0x10 {
		// full open
		if d.code == 3 {
			for i, _ := range status {
				status[i] = 1
			}
		}
		// full close
		if d.code == 4 {

			for i, _ := range status {
				status[i] = 0
			}
		}

	}

	// open, close
	if cmd == 0x06 {
		// open

		no := lightingModuleRouteNo(data[5])
		if no == -1 {
			return
		}

		switch data[3] {
		case 0x04:
			status[no] = 1
		case 0x05:
			status[no+8] = 1
		case 0x02:
			status[no] = 0
		case 0x03:
			status[no+8] = 0
		}

	}

	// query result
	if cmd == 0x03 {

		// route 1-8
		v := data[4]

		for i := 0; i < 8; i++ {

			flag := 1 << i
			if (v & byte(flag)) == byte(flag) {
				status[7-i] = 1
			} else {
				status[7-i] = 0
			}

		}
		// todo route 9-16
		v = data[6]

		for i := 0; i < 8; i++ {

			flag := 1 << i
			if (v & byte(flag)) == byte(flag) {
				status[15-i] = 1
			} else {
				status[15-i] = 0
			}

		}
	}

	d.notifyAll()

}

func (i *LightModule16) GetId() string {
	return i.guid
}
func (i *LightModule16) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_LIGHTING_MODULE_16
}

func (i *LightModule16) GetConverter() Converter {
	return i.converter
}

func (i *LightModule16) register(o Observer) {
	i.observerList = append(i.observerList, o)
}

func (i *LightModule16) deregister(o Observer) {
	i.observerList = removeFromslice(i.observerList, o)
}

func (i *LightModule16) notifyAll() {

	p := NewPayload()

	p.Metrics = append(p.Metrics, i.status)

	for _, observer := range i.observerList {
		observer.Update(p)
	}

}

func lightingModuleRouteNo(v byte) int {

	for i := 0; i < 8; i++ {

		if (v >> i) == 1 {
			return 7 - i
		}
	}

	return -1
}

func (i *LightModule16) HeartBeat() {
	i.heart.HeartBeat()
}

func (i *LightModule16) HeartCheck() {
	i.heart.HeartCheck()
	if i.heart.Conected && i.heart.Changed() {
		i.Request("getStatus", nil)
	}
}

func (i *LightModule16) IsConnected() bool {
	return i.heart.Conected
}

func (i *LightModule16) ConnectedChanged() bool {
	return i.heart.Changed()
}

func (i *LightModule16) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.status)

	return p

}
