package core

import (
	"edge/model"
	"edge/service"
	"edge/view"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
)

type serialThings struct {
	nodeId string

	things map[uint8]Thing

	connection model.Connection

	dataChan chan<- *model.SpMessage
}

func NewSerialThings(conn model.Connection, ch chan<- *model.SpMessage, nodeId string) *serialThings {
	return &serialThings{
		nodeId: nodeId,

		things:     make(map[uint8]Thing, 256),
		connection: conn,

		dataChan: ch,
	}

}

func (s *serialThings) Create(guid string, t model.DEVICE_TYPE, addr uint8) *model.CommandResponse {

	response := &model.CommandResponse{
		Code: 200,
	}

	if addr == 0 {
		response.Code = 500
		response.Error = "addr error"

		return response
	}

	c := &model.SerialConverter{
		Addr: addr,
		Tx:   s.connection.Tx,
	}

	v := view.NewSparkPlugView(guid, s.nodeId, s.dataChan)

	thing, err := newThing(guid, t, c, v)
	if err != nil {
		response.Code = 500
		response.Error = "new device thing error"

		return response

	}

	s.things[addr] = thing

	if thing, ok := thing.(model.PassiveReportingDevice); ok {
		thing.StartLoopRequest()
	}

	err = service.DbAddSerialDevice(guid, int(addr), int(t))
	if err != nil {
		response.Code = 500
		response.Error = "save serial device error"

		return response
	}

	return response
}

func (s *serialThings) Delete(guid string) {
	for addr, thing := range s.things {
		if thing.GetId() == guid {
			delete(s.things, addr)
			service.DbDeleteSerialDevice(guid)
			return
		}
	}

}

func (s *serialThings) heartCheck() {

	for _, thing := range s.things {
		thing.HeartCheck()

		if thing.ConnectedChanged() {
			if thing.IsConnected() {

				msg := &model.SpMessage{
					Topic:   fmt.Sprintf("spBv1.0/devices/DBIRTH/%v/%v", s.nodeId, thing.GetId()),
					Payload: thing.DBirth(),
				}

				s.dataChan <- msg
			} else {
				msg := &model.SpMessage{
					Topic:   fmt.Sprintf("spBv1.0/devices/DBIRTH/%v/%v", s.nodeId, thing.GetId()),
					Payload: model.NewPayload(),
				}

				s.dataChan <- msg
			}
		}

	}
}

func (s *serialThings) heartBeatRequest() {

	for _, thing := range s.things {
		thing.Request("heartBeat", nil)
		time.Sleep(2 * time.Second)

	}

}

func (s *serialThings) Request(guid string, data []byte) {

	for _, thing := range s.things {
		if thing.GetId() == guid {
			p := model.NewPayload()
			err := proto.Unmarshal(data, p)

			if err != nil {
				return
			}

			for _, m := range p.GetMetrics() {

				cmd := *m.Name
				param := m.Value

				thing.Request(cmd, param)
			}
		}
	}

}

func (s *serialThings) Init() {
	// load serial device
}

func (s *serialThings) Process() {

	heartSendTick := time.NewTicker(120 * time.Second)
	heartCheckTick := time.NewTicker(200 * time.Second)

	for {
		select {

		case frame := <-s.connection.Rx:
			addr := frame[0]
			if thing, found := s.things[addr]; found {
				thing.Response(frame)
				thing.HeartBeat()

			}

		case <-heartSendTick.C:
			go s.heartBeatRequest()

		case <-heartCheckTick.C:
			go s.heartCheck()

		}
	}
}

// func (s *serialThings) has(guid string) bool {

// 	for _, thing := range s.things {
// 		if thing.GetId() == guid {
// 			return true
// 		}
// 	}

// 	return false
// }
