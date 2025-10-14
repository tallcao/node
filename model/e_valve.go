package model

import (
	"encoding/json"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type EValve struct {
	on bool

	guid string

	observer Observer

	Converter

	IHeart
}

func NewEValve(guid string, c Converter, o Observer) *EValve {

	item := &EValve{

		guid:      guid,
		Converter: c,

		IHeart:   new(Heart),
		observer: o,
	}

	return item
}

func (i *EValve) Request(command string, params interface{}) {

	switch command {
	case "on":
		i.SendFrame([]byte{0x01})
	case "off":
		i.SendFrame([]byte{0x00})
	case "toggle":
		if i.on {
			i.SendFrame([]byte{0x01})
		} else {
			i.SendFrame([]byte{0x00})
		}
	case "getStatus":
		i.SendFrame([]byte{0x02})
	}
}

func (i *EValve) Response(data []byte) {

	if len(data) != 8 && len(data) != 2 {
		return
	}

	on := i.on

	if len(data) == 8 {
		if data[0] == 0x00 {
			i.on = false
		}

		if data[0] == 0x01 {
			i.on = true
		}
	}

	if len(data) == 2 {
		if data[1] == 0x00 {
			i.on = false
		}

		if data[1] == 0x01 {
			i.on = true
		}
	}

	changed := (on != i.on)

	if changed {

		i.notifyAll()
	}

}

func (i *EValve) GetId() string {
	return i.guid
}
func (i *EValve) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_E_VALVE
}

func (i *EValve) notifyAll() {
	state := map[string]interface{}{
		"on": i.on,
	}

	i.observer.Update(i.guid, state)

}

func (i *EValve) HeartCheck() {
	i.IHeart.HeartCheck()
	if i.IHeart.IsConnected() && i.IHeart.ConnectedChanged() {
		i.Request("getStatus", nil)
	}
}

func (i *EValve) GetAccepted(c mqtt.Client, m mqtt.Message) {
	var desired struct {
		On bool `json:"on"`
	}

	err := json.Unmarshal(m.Payload(), &desired)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal e-valve update delta: %v", err)
		return
	}

	switch desired.On {
	case true:
		i.SendFrame([]byte{0x01})
	case false:
		i.SendFrame([]byte{0x00})
	}

}
func (i *EValve) UpdateDelta(c mqtt.Client, m mqtt.Message) {

	var desired struct {
		On bool `json:"on"`
	}

	err := json.Unmarshal(m.Payload(), &desired)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal e-valve update delta: %v", err)
		return
	}

	currentOn := i.on
	if desired.On != currentOn {
		switch desired.On {
		case true:
			i.SendFrame([]byte{0x01})
		case false:
			i.SendFrame([]byte{0x00})
		}
	}
}

func (i *EValve) CommandRequest(c mqtt.Client, m mqtt.Message) {
	var cmd CommandData

	err := json.Unmarshal(m.Payload(), &cmd)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal e-valve command: %v", err)
		return
	}

	i.Request(cmd.Command, cmd.Data)
}
