package service

import (
	"edge/model"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MqttLocalService struct {
	// UnsubChan chan string
	// SubChan   chan string

	WifiChan chan *model.MqttMsg

	ZigbeeChan chan *model.MqttMsg

	mqttClient mqtt.Client

	OnConnectedCh  chan struct{}
	OnConnectedCh2 chan struct{}
}

func (s *MqttLocalService) Unsubscribe(topic string) {

	if s.mqttClient != nil {
		s.mqttClient.Unsubscribe(topic)
	}
}

func (s *MqttLocalService) Subscribe(topic string) {

	if s.mqttClient != nil {
		s.mqttClient.Subscribe(topic, 0, s.updateCallback)
	}
}

func (s *MqttLocalService) Publish(topic string, qos byte, retained bool, payload interface{}) {

	if s.mqttClient != nil {
		s.mqttClient.Publish(topic, qos, retained, payload)
	}
}

func (s *MqttLocalService) updateCallback(c mqtt.Client, m mqtt.Message) {
	msg := &model.MqttMsg{
		Topic:   m.Topic(),
		Payload: m.Payload(),
	}
	go func() {

		if strings.HasPrefix(m.Topic(), "zigbee2mqtt/") {
			s.ZigbeeChan <- msg

		} else {
			s.WifiChan <- msg

		}

	}()
}

func (s *MqttLocalService) Run() {

	opts := mqtt.NewClientOptions().AddBroker("tcp://127.0.0.1:1883").SetClientID("edge-mqtt").SetOrderMatters(false).SetOnConnectHandler(s.onConnectHandler)

	c := mqtt.NewClient(opts)

	token := c.Connect()

	s.mqttClient = c

	for token.Wait() && token.Error() != nil {
		time.Sleep(10 * time.Second)
		token = c.Connect()
	}

	defer c.Disconnect(250)

	select {}

}

func (s MqttLocalService) onConnectHandler(c mqtt.Client) {
	go func() {
		s.OnConnectedCh <- struct{}{}
		s.OnConnectedCh2 <- struct{}{}
	}()
}
