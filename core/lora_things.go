package core

import (
	"edge/model"
	"edge/service"
	"edge/view"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type loraThings struct {
	nodeID     string
	mu         sync.Mutex
	things     map[string]model.Thing
	connection model.Connection

	pubChan chan<- *model.MqttMsg

	isParing    bool
	paringMutex sync.Mutex
	view        model.Observer

	loraPanelView model.Observer
}

func (s *loraThings) SetNodeUUID(uuid string) {
	s.nodeID = uuid
}

func (s *loraThings) UpdateDevice(device model.Device) {
	switch device.Operation {
	case "add":
		s.add(device)

	case "delete":
		s.delete(device)
	}
}

func NewLoraThings(conn model.Connection, ch chan<- *model.MqttMsg, id string) *loraThings {
	return &loraThings{
		nodeID:        id,
		things:        make(map[string]model.Thing),
		connection:    conn,
		view:          view.NewShadowView(ch),
		loraPanelView: view.NewEventView("lora-panel", ch),
		pubChan:       ch,
	}
}

func (s *loraThings) PermitJoinCallback(c mqtt.Client, m mqtt.Message) {

	var data struct {
		Time int64 `json:"time"`
	}

	err := json.Unmarshal(m.Payload(), &data)

	if err != nil {
		return
	}

	s.paringMutex.Lock()         // Lock the mutex to ensure thread safety
	defer s.paringMutex.Unlock() // Ensure the mutex is unlocked after processing
	if s.isParing {
		return
	}
	s.isParing = true

	go func() {
		time.Sleep(time.Second * time.Duration(data.Time))
		s.paringMutex.Lock()
		s.isParing = false
		s.paringMutex.Unlock()
	}()

}

func (s *loraThings) add(device model.Device) error {

	index := device.ConverterSN
	if thing, found := s.things[index]; found {
		if thing, ok := thing.(model.PassiveReportingDevice); ok {
			thing.StopLoopRequest()
		}
	}

	id, err := hex.DecodeString(device.ConverterSN)

	if err != nil {
		return err
	}
	cmd := getLoraCmd(device.ConverterType)

	converter := &model.LoraConverter{
		SN:       device.ConverterSN,
		Id:       id,
		Cmd:      cmd,
		LoraType: device.ConverterType,
		Tx:       s.connection.Tx,
	}

	observer := s.view

	if device.Vendor == "ztnet" && device.Model == "lora_panel" {
		observer = s.loraPanelView
	}
	thing, err := newThing(device.UUID, device.Vendor, device.Model, converter, observer)
	if err != nil {
		return err
	}

	if parent, ok := thing.(model.Parent); ok {
		for _, child := range device.Children {

			deviceChild := model.NewLightModuleChild(child.UUID, child.No, s.view, thing)
			topic := fmt.Sprintf("%v/shadow/update/delta", child.UUID)
			service.GetMqttService().AddTopicHandler(topic, deviceChild.UpdateDelta)
			service.GetMqttService().AddSubscriptionTopic(topic, 1)

			topic = fmt.Sprintf("%v/shadow/get/accepted", child.UUID)
			service.GetMqttService().AddTopicHandler(topic, deviceChild.GetAccepted)
			service.GetMqttService().AddSubscriptionTopic(topic, 1)

			topic = fmt.Sprintf("commands/%v", child.UUID)
			service.GetMqttService().AddTopicHandler(topic, deviceChild.CommandRequest)
			service.GetMqttService().AddSubscriptionTopic(topic, 0)

			parent.AddChild(deviceChild)
		}

	}

	s.mu.Lock()
	s.things[index] = thing
	s.mu.Unlock()
	switch thing := thing.(type) {
	case model.Device485:
		converter.Setting485(thing.GetDevice485Setting())
	}

	if thing, ok := thing.(model.PassiveReportingDevice); ok {
		thing.StartLoopRequest()
	}

	return nil
}

func (s *loraThings) delete(device model.Device) {
	s.mu.Lock()
	if thing, ok := s.things[device.ConverterSN]; ok {

		if parent, ok := thing.(model.Parent); ok {

			for _, id := range parent.GetChildrenIds() {
				service.GetMqttService().DeleteSubscriptionTopic(fmt.Sprintf("%v/shadow/update/delta", id))
				service.GetMqttService().DeleteSubscriptionTopic(fmt.Sprintf("commands/%v", id))
				service.GetMqttService().DeleteSubscriptionTopic(fmt.Sprintf("%v/shadow/get/accepted", id))

			}
			parent.RemoveChildren()
		}
		service.GetMqttService().DeleteSubscriptionTopic(fmt.Sprintf("%v/shadow/update/delta", device.UUID))
		service.GetMqttService().DeleteSubscriptionTopic(fmt.Sprintf("commands/%v", device.UUID))
		service.GetMqttService().DeleteSubscriptionTopic(fmt.Sprintf("%v/shadow/get/accepted", device.UUID))

		if thing, ok := thing.(model.PassiveReportingDevice); ok {
			thing.StopLoopRequest()
		}

		delete(s.things, device.ConverterSN)

	}

	s.mu.Unlock()
}

func (s *loraThings) heartCheck() {

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, thing := range s.things {
		thing.HeartCheck()

		if thing.ConnectedChanged() {

			var data struct {
				DeviceUUID string         `json:"device_uuid"`
				State      map[string]any `json:"state"`
			}

			data.DeviceUUID = thing.GetId()
			data.State = make(map[string]any)

			data.State["connected"] = thing.IsConnected()

			if payload, err := json.Marshal(data); err == nil {
				s.pubChan <- &model.MqttMsg{
					Topic:   fmt.Sprintf("%v/shadow/update/reported", thing.GetId()),
					Payload: payload,
				}
			}

			// children connected
			if parent, ok := thing.(model.Parent); ok {
				for _, id := range parent.GetChildrenIds() {

					data.DeviceUUID = id
					if payload, err := json.Marshal(data); err == nil {

						s.pubChan <- &model.MqttMsg{
							Topic:   fmt.Sprintf("%v/shadow/update/reported", id),
							Payload: payload,
						}
					}
				}
			}

			if thing.IsConnected() {
				s.pubChan <- &model.MqttMsg{
					Topic:   fmt.Sprintf("%v/shadow/get", thing.GetId()),
					Payload: "",
				}

				if parent, ok := thing.(model.Parent); ok {
					for _, id := range parent.GetChildrenIds() {
						s.pubChan <- &model.MqttMsg{
							Topic:   fmt.Sprintf("%v/shadow/get", id),
							Payload: "",
						}
					}
				}
			}
		}
	}

}

func (s *loraThings) heartBeatRequest() {

	s.mu.Lock()
	things := maps.Clone(s.things)
	s.mu.Unlock()

	for _, thing := range things {
		thing.HeartRequest()
		time.Sleep(3 * time.Second) // 避免过快请求
	}

}

func getLoraCmd(t int) byte {
	cmd := byte(0x00)

	// 4: io, 5: 485
	switch t {
	case 4:
		cmd = 0x04
	case 5:
		cmd = 0x05
	}

	return cmd
}

func (s *loraThings) RegisterResult(sn string) {

	id, err := hex.DecodeString(sn)

	if err != nil {
		log.Printf("Failed to decode lora register converter SN %s: %v", sn, err)
		return
	}
	model.LoraRegist(id, s.connection.Tx)

}
func (s *loraThings) converterRegister(data []byte) {
	sn := strings.ToUpper(hex.EncodeToString(data[3:7]))
	t := 4 + data[2] - 0xF2

	var tmp struct {
		NodeUUID      string `json:"node_uuid"`
		SN            string `json:"sn"`
		ConverterType int    `json:"converter_type"`
	}

	tmp.NodeUUID = s.nodeID
	tmp.SN = sn
	tmp.ConverterType = int(t)

	jsonData, err := json.Marshal(tmp)
	if err != nil {
		log.Printf("Failed to marshal converter register data: %v", err)
		return
	}
	topic := fmt.Sprintf("node/%v/converters/register", s.nodeID)

	err = service.DefaultMqttService.PublishMessage(topic, 0, false, jsonData)
	if err != nil {
		log.Printf("Failed to publish converter register message: %v", err)
		return

	}
}

func (s *loraThings) Process() {

	heartSendTick := time.NewTicker(120 * time.Second)
	heartCheckTick := time.NewTicker(300 * time.Second)

	for {
		select {

		case frame := <-s.connection.Rx:
			frameLen := len(frame)

			len := frame[1]

			if len != byte(frameLen) {
				continue
			}
			cmd := frame[2]

			id := frame[3:7]
			sn := strings.ToUpper(hex.EncodeToString(id))

			switch cmd {

			// 0x85: 485, 0x83: io, 0x82: button
			case 0x85, 0x83, 0x82:
				if len > 7 {
					s.mu.Lock()
					if thing, ok := s.things[sn]; ok {

						thing.HeartBeat()
						if len > 8 {
							thing.Response(frame[7 : len-1])
						}
					}
					s.mu.Unlock()
				}
			// panel register
			case 0xF1:
				if len == 8 && s.isParing {
					// lora panel
					if cmd == 0xF1 {

						var data struct {
							NodeUUID string `json:"node_uuid"`
							SN       string `json:"sn"`
						}

						data.NodeUUID = s.nodeID
						data.SN = sn

						topic := fmt.Sprintf("node/%v/lora-panels/register", s.nodeID)

						jsonData, err := json.Marshal(data)
						if err != nil {
							log.Printf("Failed to marshal converter register data: %v", err)
							continue
						}

						err = service.DefaultMqttService.PublishMessage(topic, 0, false, jsonData)
						if err != nil {
							log.Printf("Failed to publish converter register message: %v", err)
							return

						}
					}
				}

			// 0xF2 io register, 0xF3 485 register
			case 0xF2, 0xF3:
				if len == 8 && s.isParing {
					s.converterRegister(frame)
				}

			}

		case <-heartSendTick.C:
			go s.heartBeatRequest()

		case <-heartCheckTick.C:
			go s.heartCheck()

		}
	}
}
