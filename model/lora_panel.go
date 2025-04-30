package model

import (
	"time"
)

type DeviceAction struct {
	GUID string `json:"guid"`

	Command string `json:"command"`
	Param   any    `json:"param"`
}

type LoraPanelButton struct {
	ButtonValue byte            `json:"key"`
	Actions     []*DeviceAction `json:"actions"`
}
type LoraPanel struct {
	// Guid string `json:"guid"`

	// Name string `json:"name"`
	SN string `json:"sn"`
	// Buttons []*LoraPanelButton `json:"buttons"`

	time time.Time
}

// tood
func (d *LoraPanel) Request(command string, param interface{}) {
	switch command {
	case "press":

	}

}

func (m *LoraPanel) Press(key byte) string {

	if time.Since(m.time).Microseconds() < 100 {
		return ""
	}

	m.time = time.Now()

	v := ""

	switch key {
	case 1:
		v = "1"
	case 2:
		v = "2"
	case 4:
		v = "3"
	}

	return v

}
