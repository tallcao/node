package model

import (
	"encoding/json"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Door struct {
	on bool

	guid string

	observer Observer

	Converter

	IHeart
}

func NewDoor(guid string, c Converter, o Observer) *Door {

	item := &Door{

		guid:      guid,
		Converter: c,

		IHeart: new(Heart),

		observer: o,
	}

	return item
}

func (i *Door) Request(command string, params interface{}) {

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

	}

}

func (i *Door) GetAccepted(c mqtt.Client, m mqtt.Message) {

	var update struct {
		On bool `json:"on"`
	}

	err := json.Unmarshal(m.Payload(), &update)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal door update delta: %v", err)
		return
	}

	switch update.On {
	case true:
		i.SendFrame([]byte{0x01})
	case false:
		i.SendFrame([]byte{0x00})
	}
}

func (i *Door) UpdateDelta(c mqtt.Client, m mqtt.Message) {

	var update struct {
		On bool `json:"on"`
	}

	err := json.Unmarshal(m.Payload(), &update)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal door update delta: %v", err)
		return
	}

	currentOn := i.on
	if update.On != currentOn {
		switch update.On {
		case true:
			i.SendFrame([]byte{0x01})
		case false:
			i.SendFrame([]byte{0x00})
		}
	}
}
func (i *Door) Response(data []byte) {
	if len(data) != 8 && len(data) != 2 {
		return
	}

	on := i.on

	if len(data) == 8 {
		if data[0] == 0x00 {
			i.on = true
		}

		if data[0] == 0x01 {
			i.on = false

		}
	}

	if len(data) == 2 {
		if data[0] == 0x00 {
			i.on = false
		}

		if data[0] == 0x01 {
			i.on = true
		}
	}

	changed := (on != i.on)

	if changed {

		i.notifyAll()
	}

}

func (i *Door) GetId() string {
	return i.guid
}

func (i *Door) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_DOOR
}

func (i *Door) notifyAll() {

	state := map[string]any{
		"on": i.on,
	}

	i.observer.Update(i.guid, state)

}

func (i *Door) CommandRequest(c mqtt.Client, m mqtt.Message) {
	var cmd CommandData

	err := json.Unmarshal(m.Payload(), &cmd)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal door command: %v", err)
		return
	}

	i.Request(cmd.Command, cmd.Data)
}
