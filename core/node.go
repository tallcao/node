package core

import (
	"edge/model"
	"edge/service"
	"edge/system"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type nodeConfig struct {
	ID string `json:"id"`
	// Cafile string `json:"cafile"`
	Broker string `json:"broker"`
}

type syncDevices struct {
	mu      sync.Mutex
	version int64
	// uuid: bool
	devices map[string]bool
}

type Node struct {
	id string

	uuid string

	// sparkplug
	// primaryHostAppOnline bool

	loraThings   *loraThings
	canThings    *canThings
	serialThings *serialThings

	// sparkService *service.SparkplugService

	// dataChan chan *model.SpMessage

	mqtt        *service.MqttService
	mqttPubChan chan *model.MqttMsg

	syncDevices
}

func NewNode(file string, dbus *service.DbusService) *Node {

	content, err := os.ReadFile(file)

	if err != nil {
		return nil
	}

	config := &nodeConfig{}
	err = json.Unmarshal(content, config)
	if err != nil {
		return nil
	}

	id := config.ID
	// ca := config.Cafile
	ca :=""
	uri := config.Broker

	ch := make(chan *model.MqttMsg, 100)

	service.InitMqttService(id, uri, ca)
	return &Node{

		id: id,

		canThings:    NewCanThings(dbus.CanConnection(), ch, id),
		loraThings:   NewLoraThings(dbus.LoraConnection(), ch, id),
		serialThings: NewSerialThings(dbus.SerialConnection(), ch, id),

		mqttPubChan: ch,
		mqtt:        service.GetMqttService(),

		syncDevices: syncDevices{
			devices: make(map[string]bool),
		},
	}

}

func (n *Node) devicesUpdateCallback(c mqtt.Client, m mqtt.Message) {

	var update model.DevicesUpdate
	err := json.Unmarshal(m.Payload(), &update)
	if err != nil {
		log.Printf("Failed to unmarshal devices update: %v", err)
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	// if update.Version <= n.version {
	// 	log.Printf("Ignoring outdated devices update: %d <= %d", update.Version, n.version)
	// 	return
	// }
	n.version = update.Version

	for _, device := range update.Devices {

		if device.Operation == "add" {
			n.devices[device.UUID] = true
		} else {
			delete(n.devices, device.UUID)
		}

		connectionType := strings.ToLower(device.ConnectionType)
		switch connectionType {
		case model.ConnectionTypeCAN:
			n.canThings.UpdateDevice(device)
		case model.ConnectionTypeLora:
			n.loraThings.UpdateDevice(device)
		case model.ConnectionTypeSerial:
			n.serialThings.UpdateDevice(device)
		default:
			log.Printf("Unknown device connection type: %v", connectionType)
		}
	}
}
func (n *Node) converterRegisterCallback(c mqtt.Client, m mqtt.Message) {

	var result struct {
		// Node          string `json:"node"`
		SN            string `json:"sn"`
		ConverterType int    `json:"converter_type"`
		CanID         int    `json:"can_id,omitempty"`
	}

	err := json.Unmarshal(m.Payload(), &result)
	if err != nil {
		log.Printf("Failed to unmarshal converter register result: %v", err)
		return
	}

	switch result.ConverterType {
	case 1, 2, 3: // can-io, can-485, can-relay
		n.canThings.RegisterResult(result.SN, result.CanID)
	case 0, 4, 5, 6: //lora-panel, lora-io, lora-485, lora-relay
		n.loraThings.RegisterResult(result.SN)

	}
}

func (n *Node) getUUIDCallback(c mqtt.Client, m mqtt.Message) {

	n.uuid = string(m.Payload())

	n.canThings.SetNodeUUID(n.uuid)
	n.loraThings.SetNodeUUID(n.uuid)
	n.serialThings.SetNodeUUID(n.uuid)

	topic := fmt.Sprintf("node/%v/lora/permit-join", n.uuid)
	n.mqtt.AddTopicHandler(topic, n.loraThings.PermitJoinCallback)
	n.mqtt.AddSubscriptionTopic(topic, 1)

	topic = fmt.Sprintf("node/%v/devices/update", n.uuid)
	n.mqtt.AddTopicHandler(topic, n.devicesUpdateCallback)
	n.mqtt.AddSubscriptionTopic(topic, 1)

	topic = fmt.Sprintf("node/%v/converters/register/result", n.uuid)
	n.mqtt.AddTopicHandler(topic, n.converterRegisterCallback)
	n.mqtt.AddSubscriptionTopic(topic, 1)

	n.publishDevices()
}

func (n *Node) onConnectHandler(c mqtt.Client) {
	n.mqtt.PublishMessage(fmt.Sprintf("node/%v/sn", n.id), 1, true, n.id)

	n.mqtt.PublishMessage(fmt.Sprintf("node/%v/connected", n.id), 1, true, "true")

	var info struct {
		IP      string `json:"ip"`
		SN      string `json:"sn"`
		Version string `json:"version"`
	}

	info.IP = system.IPv4()
	info.SN = n.id
	info.Version = system.NodeVersion()
	infoJSON, err := json.Marshal(info)
	if err != nil {
		log.Printf("ERROR: Failed to marshal node info: %v", err)
	} else {
		n.mqtt.PublishMessage(fmt.Sprintf("node/%v/info", n.id), 1, true, infoJSON)
	}

}

func (n *Node) publishDevices() {

	if n.uuid == "" {
		return
	}
	devices := make([]string, len(n.devices))
	n.mu.Lock()

	for uuid := range n.devices {
		devices = append(devices, uuid)
	}

	n.mu.Unlock()

	devicesJSON, err := json.Marshal(devices)
	if err != nil {
		log.Printf("ERROR: Failed to marshal devices: %v", err)
		return
	}
	n.mqtt.PublishMessage(fmt.Sprintf("node/%v/devices", n.uuid), 1, false, devicesJSON)
}

func (n *Node) Run() {

	go n.canThings.Process()
	go n.loraThings.Process()
	go n.serialThings.Process()

	n.mqtt.AddConnectHandler(n.onConnectHandler)

	topic := fmt.Sprintf("node/%v/uuid", n.id)
	n.mqtt.AddTopicHandler(topic, n.getUUIDCallback)
	n.mqtt.AddSubscriptionTopic(topic, 1)

	go service.GetMqttService().Start()
	defer service.GetMqttService().Stop()

	syncTick := time.NewTicker(24 * time.Hour)

	for {
		select {
		case <-syncTick.C:
			n.publishDevices()

		case msg := <-n.mqttPubChan:
			n.mqtt.PublishMessage(msg.Topic, msg.Qos, msg.Retained, msg.Payload)
		}
	}

}
