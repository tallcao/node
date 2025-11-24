package model

import (
	"edge/utils"
	"encoding/json"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// stb3-125/R 不带计量断路器

type Breaker_STB3_125_R struct {
	lock bool
	on   bool

	guid string

	addr     byte
	observer Observer

	Converter

	IHeart
}

func NewBreaker_STB3_125_R(guid string, c Converter, o Observer) *Breaker_STB3_125_R {

	item := &Breaker_STB3_125_R{

		guid: guid,

		Converter: c,

		IHeart:   new(Heart),
		observer: o,
	}

	if adapter, ok := c.(AddrAdapter); ok {
		item.addr = adapter.GetAddr()
	} else {
		item.addr = 0x01
	}

	return item
}

func (i *Breaker_STB3_125_R) Request(command string, params interface{}) {

	data := []byte{i.addr}

	switch command {
	case "heartBeat":
		data = append(data, 0x04, 0x00, 0x00, 0x00, 0x01)
	case "getStatus":
		data = append(data, 0x04, 0x00, 0x00, 0x00, 0x01)
	case "on":
		data = append(data, 0x05, 0x00, 0x01, 0xFF, 0x00)
	case "off":
		data = append(data, 0x05, 0x00, 0x01, 0x00, 0x00)
	case "toggle":
		v := byte(0xff)
		if i.on {
			v = 0
		}
		data = append(data, 0x05, 0x00, 0x01, v, 0x00)

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

func (d *Breaker_STB3_125_R) Response(data []byte) {

	if len(data) < 2 {
		return
	}

	on := d.on
	lock := d.lock

	if len(data) == 8 && data[1] == 0x05 {

		if data[4] == 0xFF {
			d.on = true
		}

		if data[4] == 0x00 {
			d.lock = false
		}

	}

	if len(data) == 7 && data[1] == 0x04 {

		if data[3] == 0x00 {
			d.lock = false

		}
		if data[3] == 0x01 {
			d.lock = true

		}
		if data[4] == 0xF0 {
			d.on = true

		}
		if data[4] == 0x0F {
			d.on = false

		}

	}

	onChanged := (on != d.on)
	lockChanged := (lock != d.lock)

	if onChanged || lockChanged {
		d.notifyAll()
	}

}

func (i *Breaker_STB3_125_R) GetId() string {
	return i.guid
}
func (i *Breaker_STB3_125_R) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_BREAKER_STB3_125_R
}

func (i *Breaker_STB3_125_R) notifyAll() {

	state := map[string]any{
		"on":   i.on,
		"lock": i.lock,
	}

	i.observer.Update(i.guid, state)

}

func (i *Breaker_STB3_125_R) HeartCheck() {
	i.IHeart.HeartCheck()
	if i.IHeart.IsConnected() && i.IHeart.ConnectedChanged() {
		i.Request("getStatus", nil)
		time.Sleep(3 * time.Second)
		i.Request("getQuantity", nil)
	}
}
func (i *Breaker_STB3_125_R) GetAccepted(c mqtt.Client, m mqtt.Message) {
	var desired struct {
		On bool `json:"on"`
	}

	err := json.Unmarshal(m.Payload(), &desired)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal breaker stb3-125-r update delta: %v", err)
		return
	}

	data := []byte{i.addr}
	switch desired.On {
	case true:
		data = append(data, 0x05, 0x00, 0x01, 0xFF, 0x00)
	case false:
		data = append(data, 0x05, 0x00, 0x01, 0x00, 0x00)
	}

	crc, err := utils.CRC16(data)
	if err != nil {
		return
	}
	data = append(data, crc...)

	i.SendFrame(data)

}
func (i *Breaker_STB3_125_R) UpdateDelta(c mqtt.Client, m mqtt.Message) {

	var desired struct {
		On bool `json:"on"`
	}

	err := json.Unmarshal(m.Payload(), &desired)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal breaker stb3-125-r update delta: %v", err)
		return
	}

	currentOn := i.on
	if desired.On != currentOn {

		data := []byte{i.addr}
		switch desired.On {
		case true:
			data = append(data, 0x05, 0x00, 0x01, 0xFF, 0x00)
		case false:
			data = append(data, 0x05, 0x00, 0x01, 0x00, 0x00)
		}

		crc, err := utils.CRC16(data)
		if err != nil {
			return
		}
		data = append(data, crc...)

		i.SendFrame(data)
	}
}

func (i *Breaker_STB3_125_R) CommandRequest(c mqtt.Client, m mqtt.Message) {
	var cmd CommandData

	err := json.Unmarshal(m.Payload(), &cmd)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal breaker stb3-125-r command: %v", err)
		return
	}

	i.Request(cmd.Command, cmd.Data)
}
