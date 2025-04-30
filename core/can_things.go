package core

import (
	"edge/model"
	"edge/service"
	"edge/view"
	"encoding/hex"
	"fmt"
	"strings"
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

	things map[uint8]Thing

	connection model.Connection

	convertersCache map[string]*canConverter

	dataChan chan<- *model.SpMessage

	view model.Observer
}

func NewCanThings(conn model.Connection, ch chan<- *model.SpMessage, nodeId string) *canThings {
	return &canThings{

		nodeId:     nodeId,
		things:     make(map[uint8]Thing, 128),
		connection: conn,

		convertersCache: make(map[string]*canConverter, 128),

		dataChan: ch,

		view: view.NewSparkPlugView(nodeId, ch),
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

	err = service.DbAddConverterDevice(guid, int(t), sn)
	if err != nil {
		response.Code = 500
		response.Error = "save can device error"

		return response
	}

	return response

}

func (s *canThings) Delete(guid string) {

	for no, thing := range s.things {
		if thing.GetId() == guid {
			delete(s.things, no)
			service.DbDeleteConverterDevice(guid)
			return
		}
	}

}

func canConverterBirthPayload(t uint32) *model.Payload {

	ts := uint64(time.Now().UnixMicro())
	typeMetric := model.NewConverterTypeMetric(t, ts)

	// noMetric := &model.Payload_Metric{}
	// *noMetric.Name = "no"
	// *noMetric.Timestamp = ts
	// *noMetric.Datatype = uint32(model.DataType_Int16)
	// noMetric.Value = &model.Payload_Metric_IntValue{IntValue: uint32(no)}

	p := model.NewPayload()
	p.Metrics = append(p.Metrics, typeMetric)

	return p

}

func (s *canThings) heartCheck() {

	for _, thing := range s.things {
		thing.HeartCheck()

		if thing.ConnectedChanged() {
			converter := s.convertersCache[thing.GetConverter().GetSN()]

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

				s.dataChan <- &model.SpMessage{
					Topic:   fmt.Sprintf("spBv1.0/converters/DBIRTH/%v/%v", s.nodeId, converter.sn),
					Payload: canConverterBirthPayload(uint32(converter.converterType)),
				}

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

				s.dataChan <- &model.SpMessage{
					Topic:   fmt.Sprintf("spBv1.0/converters/DDEATH/%v/%v", s.nodeId, converter.sn),
					Payload: model.NewPayload(),
				}

			}
		}

	}
}

func (s *canThings) heartBeatRequest() {

	for _, thing := range s.things {
		if c, ok := thing.GetConverter().(*model.CanConverter); ok {
			c.HeartRequest()
		}
		time.Sleep(2 * time.Second)

	}

}

func (s *canThings) Request(guid string, data []byte) {

	for _, thing := range s.things {
		if thing.GetId() == guid {
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

	s.things[convertNo] = thing

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
	if converters, err := service.DbGetConverters("can"); err == nil {
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
			if _, err := service.DbAddConverter(converter); err == nil {

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

				if thing, found := s.things[no]; found {
					thing.Response(data)
				}

			}

			// heart beat
			if code == 4 {
				if thing, found := s.things[no]; found {
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
