package model

type MqttMsg struct {
	Topic    string
	Payload  []byte
	Qos      byte
	Retained bool
}
