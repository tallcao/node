package model

import (
	"encoding/json"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Light struct {
	on bool

	guid string

	observer Observer

	Converter

	IHeart
}

func NewLight(guid string, c Converter, o Observer) *Light {

	item := &Light{

		guid:      guid,
		Converter: c,

		IHeart: new(Heart),

		observer: o,
	}

	return item
}

func (i *Light) Request(command string, params interface{}) {

	switch command {
	case "on":
		i.SendFrame([]byte{0x01})
	case "off":
		i.SendFrame([]byte{0x00})
	case "toggle":
		if i.on {
			i.SendFrame([]byte{0x00})
		} else {
			i.SendFrame([]byte{0x01})
		}
	}

}

func (i *Light) GetAccepted(c mqtt.Client, m mqtt.Message) {
	var desired struct {
		On bool `json:"on"`
	}

	err := json.Unmarshal(m.Payload(), &desired)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal light update delta: %v", err)
		return
	}

	switch desired.On {
	case true:
		i.SendFrame([]byte{0x01})
	case false:
		i.SendFrame([]byte{0x00})
	}

}

func (i *Light) UpdateDelta(c mqtt.Client, m mqtt.Message) {

	var desired struct {
		On bool `json:"on"`
	}

	err := json.Unmarshal(m.Payload(), &desired)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal light update delta: %v", err)
		return
	}

	if desired.On != i.on {
		switch desired.On {
		case true:
			i.SendFrame([]byte{0x01})
		case false:
			i.SendFrame([]byte{0x00})
		}
	}
}
func (i *Light) Response(data []byte) {

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

// func (i *Light) Turn(on bool) {

// 	if on {
// 		i.on()
// 	} else {
// 		i.off()
// 	}
// }
// func (i *Light) on() {
// 	i.converter.SendFrame([]byte{0x01})
// }

// func (i *Light) off() {
// 	i.converter.SendFrame([]byte{0x00})
// }

func (i *Light) GetId() string {
	return i.guid
}

func (i *Light) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_LIGHT
}

func (i *Light) notifyAll() {

	state := map[string]interface{}{
		"on": i.on,
	}

	i.observer.Update(i.guid, state)
}

func (i *Light) CommandRequest(c mqtt.Client, m mqtt.Message) {
	var cmd CommandData

	err := json.Unmarshal(m.Payload(), &cmd)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal light command: %v", err)
		return
	}

	i.Request(cmd.Command, cmd.Data)
}
