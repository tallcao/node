package model

import (
	"edge/utils"
	"encoding/binary"
	"encoding/json"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// 电表

type ElectricMeter struct {
	on bool

	energy float32

	guid string

	addr     byte
	observer Observer

	Converter

	PassiveReporting *PassiveReporting

	IHeart
}

func NewElectricMeter(guid string, c Converter, o Observer) *ElectricMeter {

	item := &ElectricMeter{

		Converter: c,
		PassiveReporting: &PassiveReporting{
			Interval: 60 * 60,
		},
		IHeart:   new(Heart),
		observer: o,
	}

	if adapter, ok := c.(AddrAdapter); ok {
		item.addr = adapter.GetAddr()
	} else {
		item.addr = 0x00
	}

	return item
}

func (i *ElectricMeter) Request(command string, params interface{}) {

	data := []byte{i.addr}

	switch command {
	case "heartBeat":
		data = append(data, 0x03, 0x00, 0x1D, 0x00, 0x02)
	case "getStatus":
		data = append(data, 0x04, 0x00, 0x64, 0x00, 0x01)
	case "getEnergy":
		data = append(data, 0x03, 0x00, 0x1D, 0x00, 0x02)
	case "on":
		data = append(data, 0x10, 0x00, 0x10, 0x00, 0x01, 0x02, 0x55, 0x55)
	case "off":
		data = append(data, 0x10, 0x00, 0x10, 0x00, 0x01, 0x02, 0xAA, 0xAA)
	case "toggle":
		v := []byte{0x55, 0x55}
		if i.on {
			v = []byte{0xAA, 0xAA}
		}
		data = append(data, 0x10, 0x00, 0x10, 0x00, 0x01, 0x02, v[0], v[1])

	default:
		return

	}
	crc, err := utils.CRC16(data)
	if err != nil {
		return
	}
	data = append(data, crc...)

	i.SendFrame(data)

}

func (d *ElectricMeter) Response(data []byte) {

	if len(data) < 7 {
		return
	}

	if len(data) == 7 && data[4] == 0x04 {

		if data[4] == 0x55 {
			d.on = true
		}
		if data[4] == 0xAA {
			d.on = false
		}

		d.notifyOn()
	}

	if len(data) == 9 && data[1] == 0x03 {

		d.energy = 0.01 * float32(binary.BigEndian.Uint32(data[3:7]))

		d.notifyEnergy()

	}

}

func (i *ElectricMeter) GetId() string {
	return i.guid
}
func (i *ElectricMeter) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_ELECTRIC_METER
}

func (i *ElectricMeter) notifyOn() {

	state := map[string]any{
		"on": i.on,
	}

	i.observer.Update(i.guid, state)

}

func (i *ElectricMeter) notifyEnergy() {

	state := map[string]any{
		"energy": i.energy,
	}

	i.observer.Update(i.guid, state)

}

func (i *ElectricMeter) HeartCheck() {
	if i.IHeart.IsConnected() && i.IHeart.ConnectedChanged() {
		i.Request("getStatus", nil)
		time.Sleep(3 * time.Second)
		i.Request("getEnergy", nil)
	}
}
func (i *ElectricMeter) GetAccepted(c mqtt.Client, m mqtt.Message) {
	var desired struct {
		On bool `json:"on"`
	}

	err := json.Unmarshal(m.Payload(), &desired)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal electric meter update delta: %v", err)
		return
	}

	data := []byte{i.addr}
	switch desired.On {
	case true:
		data = append(data, 0x10, 0x00, 0x10, 0x00, 0x01, 0x02, 0x55, 0x55)
	case false:
		data = append(data, 0x10, 0x00, 0x10, 0x00, 0x01, 0x02, 0xAA, 0xAA)
	}

	crc, err := utils.CRC16(data)
	if err != nil {
		return
	}
	data = append(data, crc...)

	i.SendFrame(data)

}
func (i *ElectricMeter) UpdateDelta(c mqtt.Client, m mqtt.Message) {

	var desired struct {
		On bool `json:"on"`
	}

	err := json.Unmarshal(m.Payload(), &desired)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal electric meter update delta: %v", err)
		return
	}

	currentOn := i.on
	if desired.On != currentOn {

		data := []byte{i.addr}
		switch desired.On {
		case true:
			data = append(data, 0x10, 0x00, 0x10, 0x00, 0x01, 0x02, 0x55, 0x55)
		case false:
			data = append(data, 0x10, 0x00, 0x10, 0x00, 0x01, 0x02, 0xAA, 0xAA)
		}

		crc, err := utils.CRC16(data)
		if err != nil {
			return
		}
		data = append(data, crc...)

		i.SendFrame(data)
	}
}

func (i *ElectricMeter) CommandRequest(c mqtt.Client, m mqtt.Message) {
	var cmd CommandData

	err := json.Unmarshal(m.Payload(), &cmd)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal electric meter command: %v", err)
		return
	}

	i.Request(cmd.Command, cmd.Data)
}

func (i *ElectricMeter) StartLoopRequest() {
	i.PassiveReporting.StartLoopRequest(i.Request, "getEnergy")
}

func (i *ElectricMeter) StopLoopRequest() {
	i.PassiveReporting.StopLoopRequest()
}

func (i *ElectricMeter) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 9600, 0, 8, 1
}
