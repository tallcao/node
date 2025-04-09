package service

type MqttService interface {
	Publish(topic string, qos byte, retained bool, payload []byte)
}
