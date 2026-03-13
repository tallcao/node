package core

import (
	"edge/model"
	"edge/service"
	"edge/view"
	"encoding/json"
	"fmt"
	"maps"
	"sync"
	"time"
)

type serialThings struct {
	nodeID string

	mu     sync.Mutex
	things map[byte]model.Thing

	connection model.Connection

	pubChan chan<- *model.MqttMsg

	view model.Observer

	serialPanelView model.Observer
}

func (s *serialThings) SetNodeUUID(uuid string) {
	s.nodeID = uuid
}

func (s *serialThings) UpdateDevice(device model.Device) {

	switch device.Operation {
	case "add":
		s.add(device)

	case "delete":
		s.delete(device)
	}
}

func NewSerialThings(conn model.Connection, ch chan<- *model.MqttMsg, nodeId string) *serialThings {
	return &serialThings{
		nodeID: nodeId,

		things:     make(map[uint8]model.Thing),
		connection: conn,

		pubChan: ch,

		view:            view.NewShadowView(ch),
		serialPanelView: view.NewEventView("serial-panel", ch),
	}

}
func (s *serialThings) add(device model.Device) error {

	index := device.Addr

	if thing, found := s.things[index]; found {
		if thing, ok := thing.(model.PassiveReportingDevice); ok {
			thing.StopLoopRequest()
		}
	}

	c := &model.SerialConverter{
		Addr: index,
		Tx:   s.connection.Tx,
	}

	observer := s.view

	if device.Vendor == "ztnet" && device.Model == "serial_panel" {
		observer = s.serialPanelView
	}

	thing, err := newThing(device.UUID, device.Vendor, device.Model, c, observer)
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

	if thing, ok := thing.(model.PassiveReportingDevice); ok {
		thing.StartLoopRequest()
	}

	return nil
}
func (s *serialThings) delete(device model.Device) {
	s.mu.Lock()

	if thing, ok := s.things[device.Addr]; ok {

		if parent, ok := thing.(model.Parent); ok {

			for _, id := range parent.GetChildrenIds() {
				service.GetMqttService().DeleteSubscriptionTopic(fmt.Sprintf("%v/shadow/update/delta", id))
				service.GetMqttService().DeleteSubscriptionTopic(fmt.Sprintf("commands/%v", id))
				service.GetMqttService().DeleteSubscriptionTopic(fmt.Sprintf("%v/shadow/get/accepted", id))

			}
			parent.RemoveChildren()

		}

		service.GetMqttService().DeleteSubscriptionTopic(fmt.Sprintf("%v/shadow/update/delta", device.UUID))
		service.GetMqttService().DeleteSubscriptionTopic(fmt.Sprintf("%v/shadow/get/accepted", device.UUID))
		service.GetMqttService().DeleteSubscriptionTopic(fmt.Sprintf("commands/%v", device.UUID))

		if thing, ok := thing.(model.PassiveReportingDevice); ok {
			thing.StopLoopRequest()
		}

		delete(s.things, device.Addr)

	}

	s.mu.Unlock()
}

func (s *serialThings) heartCheck() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, thing := range s.things {
		thing.HeartCheck()

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

func (s *serialThings) heartBeatRequest() {

	s.mu.Lock()
	things := maps.Clone(s.things)
	s.mu.Unlock()

	for _, thing := range things {
		thing.Request("heartBeat", nil)
		time.Sleep(2 * time.Second) // 避免过快请求
	}

}

func (s *serialThings) Process() {

	heartSendTick := time.NewTicker(120 * time.Second)
	heartCheckTick := time.NewTicker(200 * time.Second)

	for {
		select {

		case frame := <-s.connection.Rx:
			addr := frame[0]
			s.mu.Lock()
			if thing, ok := s.things[addr]; ok {
				thing.Response(frame)
				thing.HeartBeat()

			}
			s.mu.Unlock()

		case <-heartSendTick.C:
			go s.heartBeatRequest()

		case <-heartCheckTick.C:
			go s.heartCheck()

		}
	}
}
