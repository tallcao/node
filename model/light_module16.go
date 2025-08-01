package model

import (
	"edge/utils"
	"encoding/json"
	"log"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type LightModule16 struct {
	status uint16

	addr byte

	code int

	guid string

	observer Observer

	Converter

	IHeart
	children []*LightModuleChild
}

func NewLightModule16(guid string, c Converter, o Observer) *LightModule16 {

	item := &LightModule16{

		guid: guid,

		Converter: c,

		IHeart: new(Heart),

		observer: o,
		children: make([]*LightModuleChild, 16),
	}

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

		str := params.(string)
		i, err := strconv.ParseInt(str, 10, 64)

		if err != nil {
			return
		}

		if i < 0 || i > 15 {
			return
		}

		var cmd byte

		if command == "off" {
			cmd = 0x00
		}
		if command == "on" {
			cmd = 0x02
		}

		status := d.status
		if command == "toggle" {
			cmd = 0x00
			mask := 1 << i
			if (status & uint16(mask)) == 0 {
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

func (d *LightModule16) Response(data []byte) {

	if len(data) != 8 && len(data) != 9 {
		return
	}

	cmd := data[1]

	old := d.status

	// full open, full close
	if cmd == 0x10 {
		// full open
		if d.code == 3 {
			d.status = 0xFFFF

			for _, child := range d.children {
				child.Set(true)
			}
		}
		// full close
		if d.code == 4 {

			d.status = 0x0000

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
			d.status = v

			d.children[no].Set(true)
		case 0x02:
			v := (0xFFFF - 1<<no) & old
			d.status = v

			d.children[no].Set(false)

		case 0x05:
			v := (1 << (no + 8)) | old
			d.status = v

			d.children[no+8].Set(true)

		case 0x03:
			v := (0xFFFF - 1<<(no+8)) & old
			d.status = v

			d.children[no+8].Set(false)
		}

	}

	// query result
	if cmd == 0x03 {

		// route 1-8
		v := data[4]

		statusValue := uint16(0)

		for i := 0; i < 8; i++ {

			flag := 1 << i
			if (v & byte(flag)) == byte(flag) {

				statusValue += 1 << (7 - i)
				d.children[7-i].Set(true)
			} else {
				d.children[7-i].Set(false)
			}

		}
		// todo route 9-16
		v = data[6]

		for i := 0; i < 8; i++ {

			flag := 1 << i
			if (v & byte(flag)) == byte(flag) {
				statusValue += 1 << (15 - i)
				d.children[15-i].Set(true)

			} else {
				d.children[15-i].Set(false)
			}

		}

		d.status = statusValue

	}

	changed := (old != d.status)

	if changed {
		d.notifyAll()
	}

}

func (i *LightModule16) GetId() string {
	return i.guid
}
func (i *LightModule16) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_LIGHTING_MODULE_16
}

func (i *LightModule16) notifyAll() {

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

// todo 8-16 route
func lightingModuleRouteNo(v byte) int {

	for i := 0; i < 8; i++ {

		if (v >> i) == 1 {
			return 7 - i
		}
	}

	return -1
}

func (i *LightModule16) HeartCheck() {
	i.IHeart.HeartCheck()
	if i.IHeart.IsConnected() && i.IHeart.ConnectedChanged() {
		i.Request("getStatus", nil)
	}
}

func (d *LightModule16) AddChild(child *LightModuleChild) {

	if child.no >= 0 && child.no < 16 {
		d.children[child.no] = child

	}

}

func (i *LightModule16) GetChildrenIds() []string {

	result := make([]string, 0)
	for _, child := range i.children {
		result = append(result, child.GetId())
	}

	return result
}

func (i *LightModule16) RemoveChildren() {
	for _, child := range i.children {
		child.parent = nil
	}

	i.children = make([]*LightModuleChild, 0)
}

func (i *LightModule16) CommandRequest(c mqtt.Client, m mqtt.Message) {
	var cmd CommandData

	err := json.Unmarshal(m.Payload(), &cmd)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal light module-16 command: %v", err)
		return
	}

	i.Request(cmd.Command, cmd.Data)
}
