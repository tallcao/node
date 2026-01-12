package model

import (
	"edge/utils"
	"encoding/json"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// stb3-125/RJ 带计量断路器

type Breaker_STB3_125_RJ struct {
	lock     bool
	on       bool
	quantity float32

	guid string

	addr     byte
	observer Observer

	Converter

	PassiveReporting *PassiveReporting

	IHeart
}

func NewBreaker_STB3_125_RJ(guid string, c Converter, o Observer) *Breaker_STB3_125_RJ {

	item := &Breaker_STB3_125_RJ{

		guid:      guid,
		Converter: c,

		PassiveReporting: &PassiveReporting{
			Interval: 60 * 60,
		},

		IHeart: new(Heart),

		observer: o,
	}

	if adapter, ok := c.(AddrAdapter); ok {
		item.addr = adapter.GetAddr()
	} else {
		item.addr = 0x01
	}

	return item
}

func (i *Breaker_STB3_125_RJ) Request(command string, params interface{}) {

	data := []byte{i.addr}

	switch command {
	case "setInterval":
		i.PassiveReporting.SetInterval(params)
		// i.notifyAll()
		return
	case "heartBeat":
		data = append(data, 0x04, 0x00, 0x00, 0x00, 0x01)
	case "getStatus":
		data = append(data, 0x04, 0x00, 0x00, 0x00, 0x01)
	case "getQuantity":
		data = append(data, 0x04, 0x00, 0x25, 0x00, 0x02)
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

func (d *Breaker_STB3_125_RJ) Response(data []byte) {

	if len(data) < 2 {
		return
	}

	on := d.on
	lock := d.lock
	quantity := d.quantity

	if len(data) == 8 && data[1] == 0x05 {

		if data[4] == 0xFF {
			d.on = true
		}

		if data[4] == 0x00 {
			d.on = false
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
			d.lock = true

		}
		if data[4] == 0x0F {
			d.lock = false

		}

	}

	if len(data) == 9 && data[1] == 0x04 {
		d.quantity = float32(data[3])*256 + float32(data[4]) + float32((int64(data[5])*256+int64(data[6]))/1000)
	}

	onChanged := (on != d.on)
	lockChanged := (lock != d.lock)
	quantityChanged := (quantity != d.quantity)

	if onChanged || lockChanged || quantityChanged {
		d.notifyAll()
	}

}

func (i *Breaker_STB3_125_RJ) GetId() string {
	return i.guid
}

func (i *Breaker_STB3_125_RJ) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_BREAKER_STB3_125_RJ
}

func (i *Breaker_STB3_125_RJ) notifyAll() {

	state := map[string]any{
		"on":       i.on,
		"lock":     i.lock,
		"quantity": i.quantity,
	}

	i.observer.Update(i.guid, state)
}

func (i *Breaker_STB3_125_RJ) StartLoopRequest() {
	i.PassiveReporting.StartLoopRequest(i.Request, "getQuantity")
}

func (i *Breaker_STB3_125_RJ) StopLoopRequest() {
	i.PassiveReporting.StopLoopRequest()
}

func (i *Breaker_STB3_125_RJ) HeartCheck() {
	i.IHeart.HeartCheck()
	if i.IHeart.IsConnected() && i.IHeart.ConnectedChanged() {
		i.Request("getStatus", nil)
		time.Sleep(3 * time.Second)
		i.Request("getQuantity", nil)
	}
}
func (i *Breaker_STB3_125_RJ) GetAccepted(c mqtt.Client, m mqtt.Message) {
	var desired struct {
		On bool `json:"on"`
	}

	err := json.Unmarshal(m.Payload(), &desired)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal breaker stb3-125-RJ update delta: %v", err)
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
func (i *Breaker_STB3_125_RJ) UpdateDelta(c mqtt.Client, m mqtt.Message) {

	var desired struct {
		On bool `json:"on"`
	}

	err := json.Unmarshal(m.Payload(), &desired)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal breaker stb3-125-RJ update delta: %v", err)
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

func (i *Breaker_STB3_125_RJ) CommandRequest(c mqtt.Client, m mqtt.Message) {
	var cmd CommandData

	err := json.Unmarshal(m.Payload(), &cmd)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal breaker stb3-125-rj command: %v", err)
		return
	}

	i.Request(cmd.Command, cmd.Data)
}
