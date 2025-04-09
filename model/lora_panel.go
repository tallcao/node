package model

import (
	"time"
)

type DeviceAction struct {
	GUID string `json:"guid"`

	Command string      `json:"command"`
	Param   interface{} `json:"param"`
}

type LoraPanelButton struct {
	ButtonValue byte            `json:"key"`
	Actions     []*DeviceAction `json:"actions"`
}
type LoraPanel struct {
	// Guid string `json:"guid"`

	Name    string             `json:"name"`
	SN      string             `json:"sn"`
	Buttons []*LoraPanelButton `json:"buttons"`

	time time.Time
}

// tood
func (d *LoraPanel) Request(command string, param interface{}) {
	switch command {
	case "press":

	}

}

func (m *LoraPanel) Press(key byte) []*DeviceAction {

	if time.Since(m.time).Microseconds() < 100 {
		return nil
	}

	m.time = time.Now()
	for _, btn := range m.Buttons {
		if btn.ButtonValue == key {
			return btn.Actions
		}
	}

	return nil

}
