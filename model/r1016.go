package model

import (
	"encoding/json"
	"log"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type R1016 struct {
	status int

	addr byte

	guid string

	observer Observer

	Converter

	IHeart
	children []*LightModuleChild
}

func NewR1016(guid string, c Converter, o Observer) *R1016 {

	item := &R1016{

		guid: guid,

		Converter: c,

		IHeart: new(Heart),

		observer: o,
		children: make([]*LightModuleChild, 16),
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

	old := d.status

	statusValue := 0

	for i := 0; i < 16; i++ {

		v := data[i+5]
		statusValue += int(v) << i

		d.children[i].Set(v == 0x01)

	}
	d.status = statusValue

	changed := (old != d.status)

	if changed {
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

	status := []bool{
		i.status&0x0001 == 0x0001,
		i.status&0x0002 == 0x0002,
		i.status&0x0004 == 0x0004,
		i.status&0x0008 == 0x0008,
		i.status&0x0010 == 0x0010,
		i.status&0x0020 == 0x0020,
		i.status&0x0040 == 0x0040,
		i.status&0x0080 == 0x0080,
		i.status&0x0100 == 0x0100,
		i.status&0x0200 == 0x0200,
		i.status&0x0400 == 0x0400,
		i.status&0x0800 == 0x0800,
		i.status&0x1000 == 0x1000,
		i.status&0x2000 == 0x2000,
		i.status&0x4000 == 0x4000,
		i.status&0x8000 == 0x8000,
	}

	state := map[string]interface{}{
		"status": status,
	}

	i.observer.Update(i.guid, state)

}

func (i *R1016) HeartCheck() {
	i.IHeart.HeartCheck()
	if i.IHeart.IsConnected() && i.IHeart.ConnectedChanged() {
		i.Request("getStatus", nil)
	}
}

func (d *R1016) AddChild(child *LightModuleChild) {

	if child.no >= 0 && child.no < 16 {
		d.children[child.no] = child

	}

}

func (i *R1016) GetChildrenIds() []string {

	result := make([]string, 0)
	for _, child := range i.children {
		result = append(result, child.GetId())
	}

	return result
}

func (i *R1016) RemoveChildren() {

	for _, child := range i.children {
		child.parent = nil
	}

	i.children = make([]*LightModuleChild, 0)
}

func (i *R1016) CommandRequest(c mqtt.Client, m mqtt.Message) {
	var cmd CommandData

	err := json.Unmarshal(m.Payload(), &cmd)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal r1016 command: %v", err)
		return
	}

	i.Request(cmd.Command, cmd.Data)
}
