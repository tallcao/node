package model

type MqttMsg struct {
	Topic    string
	Payload  any
	Qos      byte
	Retained bool
}
