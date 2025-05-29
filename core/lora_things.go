package core

import (
	"database/sql"
	"edge/model"
	"edge/service"
	"edge/view"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

type loraConverter struct {
	sn            string
	converterType int
}

// type panels struct {
// 	Panels    []*model.LoraPanel
// 	Timestamp int64
// }

type loraThings struct {
	nodeId string

	// converter sn ->thing
	things sync.Map

	connection model.Connection

	// key: sn
	// panels map[string]*model.LoraPanel

	// ActionChan chan *model.DeviceAction

	converterLoraCache map[string]*loraConverter
	// converterPanelCache []*loraPanel
	// converterPanelCache map[string]*loraPanel

	isParing bool

	dataChan chan<- *model.SpMessage

	view model.Observer

	db *sql.DB
}

func NewLoraThings(conn model.Connection, ch chan<- *model.SpMessage, nodeId string, db *sql.DB) *loraThings {
	return &loraThings{
		nodeId:     nodeId,
		connection: conn,
		// updatePanelsChan: make(chan []byte),

		// ActionChan: make(chan *model.DeviceAction),
		// things: make(map[string]model.Thing, 128),

		converterLoraCache: make(map[string]*loraConverter, 128),
		// converterPanelCache: make([]*loraPanel, 0, 64),
		// converterPanelCache: make(map[string]*loraPanel, 64),

		dataChan: ch,

		view: view.NewSparkPlugView(nodeId, ch),

		db: db,
	}

}

func (s *loraThings) addLoraThing(guid string, deviceType int, converterSN string, converterType int) error {

	id, err := hex.DecodeString(converterSN)

	if err != nil {
		return err
	}
	cmd := getLoraCmd(converterType)

	converter := &model.LoraConverter{
		SN:       converterSN,
		Id:       id,
		Cmd:      cmd,
		LoraType: converterType,
		Tx:       s.connection.Tx,
	}

	thing, err := newThing(guid, model.DEVICE_TYPE(deviceType), converter, s.view)
	if err != nil {
		return err
	}
	s.things.Store(converterSN, thing)

	switch thing := thing.(type) {
	case model.Device485:
		converter.Setting485(thing.GetDevice485Setting())
	}

	if thing, ok := thing.(model.PassiveReportingDevice); ok {
		thing.StartLoopRequest()
	}

	return nil
}

func (s *loraThings) Create(guid string, t model.DEVICE_TYPE, sn string) *model.CommandResponse {

	response := &model.CommandResponse{
		Code: 200,
	}

	if sn == "" {
		response.Code = 500
		response.Error = "no sn value"

		return response
	}

	converter, found := s.converterLoraCache[sn]
	if !found {
		response.Code = 500
		response.Error = "device of sn value no found"

		return response
	}

	err := s.addLoraThing(guid, int(t), converter.sn, converter.converterType)
	if err != nil {
		response.Code = 500
		response.Error = "add can device error"

		return response
	}

	err = service.DbAddConverterDevice(s.db, guid, int(t), sn)
	if err != nil {
		response.Code = 500
		response.Error = "save can device error"

		return response
	}

	return response

}

func (s *loraThings) Delete(guid string) {

	s.things.Range(func(key, value any) bool {

		if thing, ok := value.(model.Thing); ok {
			if thing.GetId() == guid {
				s.things.Delete(key)
				service.DbDeleteConverterDevice(s.db, guid)
				return false
			}
		}
		return true
	})

}

func loraConverterBirthPayload(t int) *model.Payload {

	ts := uint64(time.Now().UnixMicro())
	typeMetric := model.NewConverterTypeMetric(uint32(t), ts)

	p := model.NewPayload()
	p.Metrics = append(p.Metrics, typeMetric)

	return p

}

func (s *loraThings) heartCheck() {

	s.things.Range(func(key, value any) bool {
		if thing, ok := value.(model.Thing); ok {
			thing.HeartCheck()

			if thing.ConnectedChanged() {
				// converter := s.converterLoraCache[thing.GetConverter().GetSN()]

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

					// s.dataChan <- &model.SpMessage{
					// 	Topic:   fmt.Sprintf("spBv1.0/converters/DBIRTH/%v/%v", s.nodeId, converter.sn),
					// 	Payload: loraConverterBirthPayload(converter.converterType),
					// }
				} else {
					s.dataChan <- &model.SpMessage{
						Topic:   fmt.Sprintf("spBv1.0/devices/DDEATH/%v/%v", s.nodeId, thing.GetId()),
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

					// s.dataChan <- &model.SpMessage{
					// 	Topic:   fmt.Sprintf("spBv1.0/converters/DDEATH/%v/%v", s.nodeId, converter.sn),
					// 	Payload: model.NewPayload(),
					// }
				}
			}

		}
		return true

	})

}

func (s *loraThings) heartBeatRequest() {

	s.things.Range(func(key, value interface{}) bool {
		if thing, ok := value.(model.Thing); ok {
			thing.HeartRequest()
			time.Sleep(3 * time.Second)
		}
		return true

	})

}

func (s *loraThings) Request(guid string, data []byte) {

	if thing, found := s.getThingByGuid(guid); found {

		p := &model.Payload{}
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

func (s *loraThings) addConverter(sn string, t int) {

	item := &loraConverter{
		sn:            sn,
		converterType: t,
	}
	s.converterLoraCache[sn] = item
}

func (s *loraThings) Paring(t int) {

	if s.isParing {
		return
	}
	s.isParing = true
	time.Sleep(time.Minute * time.Duration(t))
	s.isParing = false

}

func (s *loraThings) Process() {

	heartSendTick := time.NewTicker(120 * time.Second)
	heartCheckTick := time.NewTicker(300 * time.Second)

	for {
		select {

		case frame := <-s.connection.Rx:

			len := frame[1]
			cmd := frame[2]

			id := frame[3:7]
			sn := strings.ToUpper(hex.EncodeToString(id))

			switch cmd {

			// 0x85: 485, 0x83: io, 0x82: button
			case 0x85, 0x83, 0x82:
				if len > 7 {
					if value, ok := s.things.Load(sn); ok {
						thing := value.(model.Thing)
						thing.HeartBeat()
						if len > 8 {
							thing.Response(frame[7 : len-1])
						}
					}
				}
			// panel register
			case 0xF1:
				if len == 8 && s.isParing {
					model.LoraRegist(id, s.connection.Tx)

					s.dataChan <- &model.SpMessage{
						Topic: fmt.Sprintf("spBv1.0/lora-panels/DBIRTH/%v/%v", s.nodeId, sn),
						// Payload: loraPanelBirthPayload(guid),
						Payload: model.NewPayload(),
					}

				}

			//0xF2 io register, 0xF3 485 register
			case 0xF2, 0xF3:
				if len == 8 && s.isParing {
					model.LoraRegist(id, s.connection.Tx)

					t := 4 + cmd - 0xF2

					converter := &service.DbConverter{
						SN:            sn,
						ConverterType: int64(t),
					}
					if _, err := service.DbAddConverter(s.db, converter); err == nil {

						s.addConverter(sn, int(t))

						s.dataChan <- &model.SpMessage{
							Topic:   fmt.Sprintf("spBv1.0/converters/DBIRTH/%v/%v", s.nodeId, converter.SN),
							Payload: loraConverterBirthPayload(int(converter.ConverterType)),
						}

					}

				}
			}

		case <-heartSendTick.C:
			go s.heartBeatRequest()

		case <-heartCheckTick.C:
			go s.heartCheck()

		}
	}
}

func (s *loraThings) Init() {

	// load converter, devices
	if converters, err := service.DbGetConverters(s.db, "lora"); err == nil {
		for _, c := range converters {
			s.addConverter(c.SN, int(c.ConverterType))

			if c.Guid != nil && c.DeviceType != nil {
				s.addLoraThing(*c.Guid, int(*c.DeviceType), c.SN, int(c.ConverterType))
			}
		}

	}

}

func (s *loraThings) getThingByGuid(guid string) (model.Thing, bool) {

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
