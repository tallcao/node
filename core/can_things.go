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

type canConverter struct {
	id int
	sn string

	// 1: io, 2: 485, 3: relay
	converterType int
}

type canThings struct {
	nodeId string

	// can no ->thing
	things sync.Map

	connection model.Connection

	convertersCache map[string]*canConverter

	dataChan chan<- *model.SpMessage

	view model.Observer

	db *sql.DB
}

func NewCanThings(conn model.Connection, ch chan<- *model.SpMessage, nodeId string, db *sql.DB) *canThings {
	return &canThings{

		nodeId:     nodeId,
		connection: conn,

		convertersCache: make(map[string]*canConverter, 128),

		dataChan: ch,

		view: view.NewSparkPlugView(nodeId, ch),

		db: db,
	}

}

func (s *canThings) Create(guid string, t model.DEVICE_TYPE, sn string) *model.CommandResponse {

	response := &model.CommandResponse{
		Code: 200,
	}

	if sn == "" {
		response.Code = 500
		response.Error = "no sn value"

		return response
	}

	converter, found := s.convertersCache[sn]
	if !found {
		response.Code = 500
		response.Error = "device of sn value no found"

		return response
	}

	err := s.addCanThing(guid, int(t), converter.sn, uint8(converter.id), converter.converterType)
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

func (s *canThings) Delete(guid string) {

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

func canConverterBirthPayload(t uint32) *model.Payload {

	ts := uint64(time.Now().UnixMicro())
	typeMetric := model.NewConverterTypeMetric(t, ts)

	p := model.NewPayload()
	p.Metrics = append(p.Metrics, typeMetric)

	return p

}

func (s *canThings) heartCheck() {

	s.things.Range(func(key, value any) bool {
		if thing, ok := value.(model.Thing); ok {
			thing.HeartCheck()

			if thing.ConnectedChanged() {
				// converter := s.convertersCache[thing.GetConverter().GetSN()]

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
					// 	Payload: canConverterBirthPayload(uint32(converter.converterType)),
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

func (s *canThings) heartBeatRequest() {

	s.things.Range(func(key, value any) bool {
		if thing, ok := value.(model.Thing); ok {
			thing.HeartRequest()
			time.Sleep(2 * time.Second)
		}
		return true

	})

}
func (s *canThings) getThingByGuid(guid string) (model.Thing, bool) {

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

func (s *canThings) Request(guid string, data []byte) {

	if thing, found := s.getThingByGuid(guid); found {

		p := &model.Payload{}
		err := proto.Unmarshal(data, p)

		if err != nil {
			return
		}

		for _, m := range p.GetMetrics() {

			cmd := *m.Name
			param := m.GetStringValue()

			thing.Request(cmd, param)
		}
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

func (s *canThings) generateNO() (byte, error) {

	for i := 1; i < 128; i++ {

		used := false
		for _, c := range s.convertersCache {

			if c.id == i {
				used = true
				break
			}

		}
		if !used {
			return byte(i), nil
		}

	}

	return 0, fmt.Errorf("generate can no failed")
}

func (s *canThings) addCanThing(guid string, deviceType int, converterSN string, convertNo uint8, converterType int) error {

	code := getCode(converterType)

	converter := &model.CanConverter{
		SN:   converterSN,
		No:   convertNo,
		Code: code,
		Tx:   s.connection.Tx,
	}

	t := model.DEVICE_TYPE(deviceType)

	thing, err := newThing(guid, t, converter, s.view)
	if err != nil {
		return err
	}

	s.things.Store(convertNo, thing)

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

func (s *canThings) Init() {

	// load converters, devices
	if converters, err := service.DbGetConverters(s.db, "can"); err == nil {
		for _, c := range converters {
			s.addConverter(int(c.CanNo), c.SN, int(c.ConverterType))

			if c.Guid != nil && c.DeviceType != nil {
				s.addCanThing(*c.Guid, int(*c.DeviceType), c.SN, uint8(c.CanNo), int(c.ConverterType))
			}

		}

	}

}

func (s *canThings) addConverter(id int, sn string, t int) {
	item := &canConverter{
		id:            id,
		sn:            sn,
		converterType: t,
	}
	s.convertersCache[sn] = item

}

func (s *canThings) converterRegister(data []byte) {

	sn := strings.ToUpper(hex.EncodeToString(data[1:]))

	if c, found := s.convertersCache[sn]; found {
		model.CanRegist(data[1:], byte(c.id), s.connection.Tx)
	} else {
		// new converter
		if no, err := s.generateNO(); err == nil {

			converter := &service.DbConverter{
				SN:            sn,
				ConverterType: int64(data[0]),
				CanNo:         int64(no),
			}
			if _, err := service.DbAddConverter(s.db, converter); err == nil {

				s.addConverter(int(no), sn, int(data[0]))

				model.CanRegist(data[1:], no, s.connection.Tx)

				s.dataChan <- &model.SpMessage{
					Topic:   fmt.Sprintf("spBv1.0/converters/DBIRTH/%v/%v", s.nodeId, converter.SN),
					Payload: canConverterBirthPayload(uint32(converter.ConverterType)),
				}

			}
		}
	}
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

				if value, ok := s.things.Load(no); ok {
					thing := value.(model.Thing)
					thing.Response(data)
				}

			}

			// heart beat
			if code == 4 {
				if value, ok := s.things.Load(no); ok {
					thing := value.(model.Thing)
					thing.HeartBeat()
				}
			}

		case <-heartSendTick.C:
			go s.heartBeatRequest()

		case <-heartCheckTick.C:
			go s.heartCheck()

		}
	}
}
