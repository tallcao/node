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
)

type canThings struct {
	nodeId string

	mu     sync.Mutex
	things map[byte]model.Thing

	connection model.Connection

	pubChan chan<- *model.MqttMsg

	view model.Observer
}

func (s *canThings) UpdateDevice(device model.Device) {
	switch device.Operation {
	case "add":
		s.add(device)

	case "delete":
		s.delete(device)

	}
}

func NewCanThings(conn model.Connection, ch chan<- *model.MqttMsg, nodeId string) *canThings {
	return &canThings{

		nodeId:     nodeId,
		connection: conn,

		things:  make(map[byte]model.Thing),
		pubChan: ch,

		view: view.NewShadowView(nodeId, ch),
	}

}

func (s *canThings) delete(device model.Device) {

	s.mu.Lock()

	if thing, ok := s.things[device.CanID]; ok {

		if parent, ok := thing.(model.Parent); ok {

			for _, id := range parent.GetChildrenIds() {
				service.GetMqttService().DeleteSubscriptionTopic(fmt.Sprintf("%v/shadow/update/delta", id))
				service.GetMqttService().DeleteSubscriptionTopic(fmt.Sprintf("commands/%v", id))

			}
			parent.RemoveChildren()

		}

		service.GetMqttService().DeleteSubscriptionTopic(fmt.Sprintf("%v/shadow/update/delta", device.UUID))
		service.GetMqttService().DeleteSubscriptionTopic(fmt.Sprintf("commands/%v", device.UUID))

		delete(s.things, device.CanID)

	}

	s.mu.Unlock()
}

func (s *canThings) heartCheck() {

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, thing := range s.things {
		thing.HeartCheck()

		if thing.ConnectedChanged() {

			payload := map[string]bool{
				"connected": thing.IsConnected(),
			}

			s.pubChan <- &model.MqttMsg{
				Topic:   fmt.Sprintf("%v/shadow/update/reported", thing.GetId()),
				Payload: payload,
			}

			// children connected
			if parent, ok := thing.(model.Parent); ok {
				for _, id := range parent.GetChildrenIds() {
					s.pubChan <- &model.MqttMsg{
						Topic:   fmt.Sprintf("%v/shadow/update/reported", id),
						Payload: payload,
					}
				}
			}

		}
	}

}

func (s *canThings) heartBeatRequest() {

	s.mu.Lock()
	things := maps.Clone(s.things)
	s.mu.Unlock()

	for _, thing := range things {
		thing.HeartRequest()
		time.Sleep(2 * time.Second) // 避免过快请求
	}

}

func getCode(t int) int {
	code := 0

	// 1: io, 2: 485, 3: relay

	switch t {
	case 1:
		code = 2
	case 2:
		code = 3
	case 3:
		code = 2
	}

	return code
}

func (s *canThings) add(device model.Device) error {

	code := getCode(device.ConverterType)

	converter := &model.CanConverter{
		SN:   device.ConverterSN,
		No:   uint8(device.CanID),
		Code: code,
		Tx:   s.connection.Tx,
	}

	thing, err := newThing(device.UUID, device.Vendor, device.Model, converter, s.view)
	if err != nil {
		return err
	}

	if parent, ok := thing.(model.Parent); ok {

		for _, child := range device.Children {

			deviceChild := model.NewLightModuleChild(child.UUID, child.No, s.view, thing)
			topic := fmt.Sprintf("%v/shadow/update/delta", child.UUID)
			service.GetMqttService().AddTopicHandler(topic, deviceChild.UpdateDelta)
			service.GetMqttService().AddSubscriptionTopic(topic, 1)

			topic = fmt.Sprintf("commands/%v", child.UUID)
			service.GetMqttService().AddTopicHandler(topic, deviceChild.CommandRequest)
			service.GetMqttService().AddSubscriptionTopic(topic, 0)

			parent.AddChild(deviceChild)
		}

	}

	s.mu.Lock()
	s.things[byte(device.CanID)] = thing
	s.mu.Unlock()

	switch thing := thing.(type) {
	case model.Device485:
		converter.Setting485(thing.GetDevice485Setting())
		// case model.DeviceRelay:
		// 	settingRelay(context, n.Id, s.GetRelayDefaultState())
	}

	if thing, ok := thing.(model.PassiveReportingDevice); ok {
		thing.StartLoopRequest()
	}

	return nil
}

func (s *canThings) converterRegister(data []byte) {

	sn := strings.ToUpper(hex.EncodeToString(data[1:]))

	var tmp struct {
		Node          string `json:"node"`
		SN            string `json:"sn"`
		ConverterType int    `json:"converter_type"`
	}

	tmp.Node = s.nodeId
	tmp.SN = sn
	tmp.ConverterType = int(data[0])

	jsonData, err := json.Marshal(tmp)
	if err != nil {
		log.Printf("Failed to marshal converter register data: %v", err)
		return
	}
	topic := fmt.Sprintf("node/%v/converters/register", s.nodeId)

	err = service.DefaultMqttService.PublishMessage(topic, 0, false, jsonData)
	if err != nil {
		log.Printf("Failed to publish converter register message: %v", err)
		return

	}
}

func (s *canThings) RegisterResult(sn string, canID int) {

	id, err := hex.DecodeString(sn)

	if err != nil {
		log.Printf("Failed to decode can register converter SN %s: %v", sn, err)
		return
	}

	model.CanRegist(id, byte(canID), s.connection.Tx)

}

func (s *canThings) Process() {

	heartSendTick := time.NewTicker(180 * time.Second)
	heartCheckTick := time.NewTicker(200 * time.Second)

	for {
		select {

		case frame := <-s.connection.Rx:

			no := frame[0]
			code := frame[1]
			data := frame[2:]

			// register
			if code == 0 {
				s.converterRegister(data)
			}

			// 3: 485
			if code == 1 || code == 2 || code == 3 {
				s.mu.Lock()
				if thing, ok := s.things[no]; ok {
					thing.Response(data)
				}
				s.mu.Unlock()
			}

			// heart beat
			if code == 4 {
				s.mu.Lock()
				if thing, ok := s.things[no]; ok {
					thing.HeartBeat()
				}
				s.mu.Unlock()
			}

		case <-heartSendTick.C:
			go s.heartBeatRequest()

		case <-heartCheckTick.C:
			go s.heartCheck()

		}
	}
}
