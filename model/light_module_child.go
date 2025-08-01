package model

import (
	"encoding/json"
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type LightModuleChild struct {
	guid string
	no   int
	on   bool

	observer Observer

	parent Thing
}

// func (i *LightModuleChild) GetParent() Thing {
// 	return i.parent
// }

func NewLightModuleChild(guid string, no int, o Observer, parent Thing) *LightModuleChild {

	return &LightModuleChild{

		guid: guid,
		no:   no,

		observer: o,

		parent: parent,
	}

}

func (i *LightModuleChild) Set(new bool) {

	changed := new != i.on
	if changed {
		i.notifyAll()
	}
}

func (i *LightModuleChild) GetId() string {
	return i.guid
}

func (i *LightModuleChild) notifyAll() {

	state := map[string]any{
		"on": i.on,
	}
	i.observer.Update(i.guid, state)

}

func (i *LightModuleChild) UpdateDelta(c mqtt.Client, m mqtt.Message) {

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
			i.parent.Request("on", i.no)
		case false:
			i.parent.Request("off", i.no)

		}
	}
}

func (i *LightModuleChild) CommandRequest(c mqtt.Client, m mqtt.Message) {
	var cmd CommandData

	err := json.Unmarshal(m.Payload(), &cmd)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal light module child command: %v", err)
		return
	}

	i.parent.Request(cmd.Command, fmt.Sprint(i.no))
}
