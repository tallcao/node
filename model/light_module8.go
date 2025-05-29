package model

import (
	"edge/utils"
	"fmt"
	"strconv"
	"time"

	"google.golang.org/protobuf/proto"
)

type LightModule8 struct {
	status *Payload_Metric

	addr byte

	code int

	guid string

	observer Observer

	Converter

	IHeart

	children []*LightModuleChild
}

func NewLightModule8(guid string, c Converter, o Observer) *LightModule8 {

	item := &LightModule8{
		status: &Payload_Metric{
			Name:      proto.String("status"),
			Datatype:  proto.Uint32(uint32(DataType_UInt8)),
			Timestamp: proto.Uint64(0),
		},

		guid:      guid,
		Converter: c,

		IHeart:   new(Heart),
		observer: o,
		children: make([]*LightModuleChild, 8),
	}

	for i := 0; i < 8; i++ {
		id := fmt.Sprintf("%v-%v", guid, i)
		child := NewLightModuleChild(id, i, o)
		item.children[i] = child
	}

	if adapter, ok := c.(AddrAdapter); ok {
		item.addr = adapter.GetAddr()
	} else {
		item.addr = 0x01
	}

	return item
}

func (d *LightModule8) Request(command string, params interface{}) {

	var data []byte

	switch command {
	case "heartBeat":
		data = []byte{d.addr, 0x03, 0x01, 0x00, 0x00, 0x02}
	case "getStatus":
		data = []byte{d.addr, 0x03, 0x01, 0x00, 0x00, 0x02}
	case "on", "off", "toggle":

		str := params.(string)
		i, err := strconv.ParseInt(str, 10, 64)

		if err != nil {
			return
		}

		if i < 0 || i > 7 {
			return
		}

		var cmd byte

		if command == "off" {
			cmd = 0x00
		}
		if command == "on" {
			cmd = 0x02
		}

		status := d.status.GetIntValue()
		if command == "toggle" {

			cmd = 0x00
			mask := 1 << i
			if (status & uint32(mask)) == 0 {
				cmd = 0x02
			}
		}

		if i < 8 {
			cmd += 2
		} else {
			cmd += 3
		}

		subAddr := 1 << ((15 - i) % 8)

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

	d.SendFrame(data)

}

func (d *LightModule8) Response(data []byte) {

	if len(data) != 8 && len(data) != 9 {
		return
	}

	cmd := data[1]

	*d.status.Timestamp = uint64(time.Now().UnixMicro())

	old := d.status.GetIntValue()

	// full open, full close
	if cmd == 0x10 {
		// full open
		if d.code == 3 {
			d.status.Value = &Payload_Metric_IntValue{0xFF}

			for _, child := range d.children {
				child.Set(true)
			}
		}
		// full close
		if d.code == 4 {

			d.status.Value = &Payload_Metric_IntValue{0}

			for _, child := range d.children {
				child.Set(false)
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
			v := (1 << no) | old
			d.status.Value = &Payload_Metric_IntValue{v}

			d.children[no].Set(true)

		case 0x02:
			v := (0xFF - 1<<no) & old
			d.status.Value = &Payload_Metric_IntValue{v}

			d.children[no].Set(false)

		}

	}

	// query result
	if cmd == 0x03 {

		// route 1-8
		v := data[4]

		statusValue := uint32(0)

		for i := 0; i < 8; i++ {

			flag := 1 << i
			if (v & byte(flag)) == byte(flag) {
				statusValue += 1 << (7 - i)
				d.children[7-i].Set(true)
			} else {

				d.children[7-i].Set(false)

			}

		}
		d.status.Value = &Payload_Metric_IntValue{statusValue}

	}

	new := d.status.GetIntValue()

	if new != old {
		d.notifyAll()
	}

}

func (i *LightModule8) GetId() string {
	return i.guid
}
func (i *LightModule8) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_LIGHTING_MODULE_8
}

func (i *LightModule8) notifyAll() {

	p := NewPayload()

	p.Metrics = append(p.Metrics, i.status)

	i.observer.Update(i.guid, p)

}

func (i *LightModule8) HeartCheck() {
	i.IHeart.HeartCheck()
	if i.IHeart.IsConnected() && i.IHeart.ConnectedChanged() {
		i.Request("getStatus", nil)
	}
}

func (i *LightModule8) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.status)

	return p

}
