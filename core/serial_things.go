package core

import (
	"database/sql"
	"edge/model"
	"edge/service"
	"edge/view"
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

type serialThings struct {
	nodeId string

	// addr -> thing
	things sync.Map

	connection model.Connection

	dataChan chan<- *model.SpMessage

	view model.Observer

	db *sql.DB
}

func NewSerialThings(conn model.Connection, ch chan<- *model.SpMessage, nodeId string, db *sql.DB) *serialThings {
	return &serialThings{
		nodeId: nodeId,

		connection: conn,

		dataChan: ch,

		view: view.NewSparkPlugView(nodeId, ch),

		db: db,
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

	thing, err := newThing(guid, t, c, s.view)
	if err != nil {
		response.Code = 500
		response.Error = "new device thing error"

		return response

	}

	s.things.Store(addr, thing)

	if thing, ok := thing.(model.PassiveReportingDevice); ok {
		thing.StartLoopRequest()
	}

	err = service.DbAddSerialDevice(s.db, guid, int(addr), int(t))
	if err != nil {
		response.Code = 500
		response.Error = err.Error()
		return response
	}

	return response
}

func (s *serialThings) Delete(guid string) {

	s.things.Range(func(key, value any) bool {

		if thing, ok := value.(model.Thing); ok {
			if thing.GetId() == guid {
				s.things.Delete(key)
				service.DbDeleteSerialDevice(s.db, guid)
				return false
			}
		}
		return true
	})
}

func (s *serialThings) heartCheck() {

	s.things.Range(func(key, value any) bool {
		if thing, ok := value.(model.Thing); ok {
			thing.HeartCheck()

			if thing.ConnectedChanged() {
				if thing.IsConnected() {

					s.dataChan <- &model.SpMessage{
						Topic:   fmt.Sprintf("spBv1.0/devices/DBIRTH/%v/%v", s.nodeId, thing.GetId()),
						Payload: thing.DBirth(),
					}

					if parent, ok := thing.(model.Parent); ok {
						for _, child := range parent.GetChildren() {
							s.dataChan <- &model.SpMessage{
								Topic:   fmt.Sprintf("spBv1.0/devices/DBIRTH/%v/%v", s.nodeId, child.GetId()),
								Payload: child.DBirth(),
							}
						}
					}

				} else {
					s.dataChan <- &model.SpMessage{
						Topic:   fmt.Sprintf("spBv1.0/devices/DBIRTH/%v/%v", s.nodeId, thing.GetId()),
						Payload: model.NewPayload(),
					}
					if parent, ok := thing.(model.Parent); ok {
						for _, child := range parent.GetChildren() {
							s.dataChan <- &model.SpMessage{
								Topic:   fmt.Sprintf("spBv1.0/devices/DDEATH/%v/%v", s.nodeId, child.GetId()),
								Payload: model.NewPayload(),
							}
						}
					}
				}
			}
		}
		return true

	})
}

func (s *serialThings) heartBeatRequest() {

	s.things.Range(func(key, value any) bool {
		if thing, ok := value.(model.Thing); ok {
			thing.Request("heartBeat", nil)
			time.Sleep(2 * time.Second)
		}
		return true

	})

}

func (s *serialThings) getThingByGuid(guid string) (model.Thing, bool) {

	var result model.Thing
	found := false

	s.things.Range(func(key, value any) bool {
		if thing, ok := value.(model.Thing); ok {
			if thing.GetId() == guid {
				result = thing
				found = true
				return false
			}

		}

		return true
	})

	return result, found
}

func (s *serialThings) Request(guid string, data []byte) {

	if thing, found := s.getThingByGuid(guid); found {

		p := &model.Payload{}
		err := proto.Unmarshal(data, p)

		if err != nil {
			return
		}

		for _, m := range p.GetMetrics() {

			cmd := m.GetName()
			param := m.GetStringValue()

			thing.Request(cmd, param)
		}
	}

}

func (s *serialThings) Init() {
	// load serial device
	if serials, err := service.DbGetSerials(s.db); err == nil {
		for _, serial := range serials {

			if serial.Guid != nil && serial.DeviceType != nil {

				addr := uint8(serial.Addr)
				t := model.DEVICE_TYPE(*serial.DeviceType)
				c := &model.SerialConverter{
					Addr: addr,
					Tx:   s.connection.Tx,
				}
				thing, err := newThing(*serial.Guid, t, c, s.view)

				if err != nil {
					continue
				}

				s.things.Store(addr, thing)
			}
		}
	}

}

func (s *serialThings) Process() {

	heartSendTick := time.NewTicker(120 * time.Second)
	heartCheckTick := time.NewTicker(200 * time.Second)

	for {
		select {

		case frame := <-s.connection.Rx:
			addr := frame[0]
			if value, ok := s.things.Load(addr); ok {
				thing := value.(model.Thing)
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
