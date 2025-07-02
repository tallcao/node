package model

import mqtt "github.com/eclipse/paho.mqtt.golang"

type Shadow interface {
	UpdateDelta(mqtt.Client, mqtt.Message)
}
