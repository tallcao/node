package model

import (
	"fmt"
	"strconv"
	"time"

	"google.golang.org/protobuf/proto"
)

type R1016 struct {
	status *Payload_Metric

	addr byte

	guid string

	observer Observer

	Converter

	IHeart
	children []*LightModuleChild
}

func NewR1016(guid string, c Converter, o Observer) *R1016 {

	item := &R1016{
		status: &Payload_Metric{
			Name:      proto.String("status"),
			Datatype:  proto.Uint32(uint32(DataType_UInt16)),
			Timestamp: proto.Uint64(0),
		},

		guid: guid,

		Converter: c,

		IHeart: new(Heart),

		observer: o,
		children: make([]*LightModuleChild, 16),
	}

	for i := 0; i < 16; i++ {

		id := fmt.Sprintf("%v-%v", guid, i)
		child := NewLightModuleChild(id, i, o)
		item.children[i] = child
	}

	if adapter, ok := c.(AddrAdapter); ok {
		item.addr = adapter.GetAddr()
	} else {
		item.addr = 0xFF
	}

	return item
}

func (d *R1016) Request(command string, params any) {

	var data []byte

	addr := []byte{d.addr, 0xFF, 0xFF, 0xFF, 0xFF}

	switch command {
	case "heartBeat":
		data = append(addr, 0x86, 0x86, 0x86, 0x0F)
	case "getStatus":
		data = append(addr, 0x86, 0x86, 0x86, 0x0F)
	case "on", "off", "toggle", "delay":

		str := params.(string)
		i, err := strconv.ParseInt(str, 10, 64)

		if err != nil {
			return
		}

		if i < 0 || i > 15 {
			return
		}

		var cmd byte

		switch command {
		case "on":
			cmd = 0x01
		case "off":
			cmd = 0x00
		case "toggle":
			cmd = 0x02
		case "delay":
			cmd = 0x03
		}

		data = append(addr, 0x00, byte(i), cmd, 0x0F)

	case "fullOn1to8":
		data = append(addr, 0x01, 0x77, 0x88, 0x0F)
	case "fullOff1to8":
		data = append(addr, 0x02, 0x77, 0x88, 0x0F)
	case "fullOn9to16":
		data = append(addr, 0x10, 0x77, 0x88, 0x0F)
	case "fullOff9to16":
		data = append(addr, 0x20, 0x77, 0x88, 0x0F)
	case "fullOn":
		data = append(addr, 0x30, 0x77, 0x88, 0x0F)
	case "fullOff":
		data = append(addr, 0x40, 0x77, 0x88, 0x0F)
	default:
		return
	}

	d.SendFrame(data)

	time.Sleep(100 * time.Millisecond)
	data = append(addr, 0x86, 0x86, 0x86, 0x0F)
	d.SendFrame(data)

}

func (d *R1016) Response(data []byte) {

	if len(data) != 22 {
		return
	}

	*d.status.Timestamp = uint64(time.Now().UnixMicro())

	old := d.status.GetIntValue()

	statusValue := uint32(0)

	for i := 0; i < 16; i++ {

		v := data[i+5]
		statusValue += uint32(v << i)

		d.children[i].Set(v == 0x01)

	}
	d.status.Value = &Payload_Metric_IntValue{statusValue}

	new := d.status.GetIntValue()

	if new != old {
		d.notifyAll()
	}

}

func (i *R1016) GetId() string {
	return i.guid
}
func (i *R1016) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_R1016
}

func (i *R1016) notifyAll() {

	p := NewPayload()

	p.Metrics = append(p.Metrics, i.status)

	i.observer.Update(i.guid, p)

}

func (i *R1016) HeartCheck() {
	i.IHeart.HeartCheck()
	if i.IHeart.IsConnected() && i.IHeart.ConnectedChanged() {
		i.Request("getStatus", nil)
	}
}

func (i *R1016) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.status)

	return p

}
