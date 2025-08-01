package model

import mqtt "github.com/eclipse/paho.mqtt.golang"

type Command interface {
	CommandRequest(mqtt.Client, mqtt.Message)
}

type CommandData struct {
	// DeviceUUID string `json:"device_uuid"`
	Command string `json:"command"`
	Data    string `json:"data,omitempty"`
}
