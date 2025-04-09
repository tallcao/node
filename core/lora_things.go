package core

import (
	"edge/model"
	"edge/service"
	"edge/view"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
)

type loraConverter struct {
	sn            string
	converterType int
}

type loraPanel struct {
	sn string
}

type panels struct {
	Panels    []*model.LoraPanel
	Timestamp int64
}

type loraThings struct {
	nodeId string

	things map[string]Thing

	connection model.Connection

	// key: sn
	panels map[string]*model.LoraPanel

	ActionChan chan *model.DeviceAction

	converterLoraCache map[string]*loraConverter
	// converterPanelCache []*loraPanel
	converterPanelCache map[string]*loraPanel

	isParing bool

	dataChan chan<- *model.SpMessage
}

func NewLoraThings(conn model.Connection, ch chan<- *model.SpMessage, nodeId string) *loraThings {
	return &loraThings{
		nodeId:     nodeId,
		things:     make(map[string]Thing, 128),
		connection: conn,
		panels:     make(map[string]*model.LoraPanel),
		// updatePanelsChan: make(chan []byte),
		// panelFile:        "/home/root/edge/panels.json",

		ActionChan: make(chan *model.DeviceAction),

		converterLoraCache: make(map[string]*loraConverter, 128),
		// converterPanelCache: make([]*loraPanel, 0, 64),
		converterPanelCache: make(map[string]*loraPanel, 64),

		dataChan: ch,
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

	v := view.NewSparkPlugView(guid, s.nodeId, s.dataChan)
	thing, err := newThing(guid, model.DEVICE_TYPE(deviceType), converter, v)
	if err != nil {
		return err
	}
	s.things[converterSN] = thing

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

	err = service.DbAddConverterDevice(guid, int(t), sn)
	if err != nil {
		response.Code = 500
		response.Error = "save can device error"

		return response
	}

	return response

}

func (s *loraThings) Delete(guid string) {

	for sn, thing := range s.things {
		if thing.GetId() == guid {
			delete(s.things, sn)
			service.DbDeleteConverterDevice(guid)
			return
		}
	}

}

func loraConverterBirthPayload(t int) *model.Payload {

	ts := uint64(time.Now().UnixMicro())
	typeMetric := model.NewConverterTypeMetric(uint32(t), ts)

	p := model.NewPayload()
	p.Metrics = append(p.Metrics, typeMetric)

	return p

}

func (s *loraThings) heartCheck() {

	for _, thing := range s.things {
		thing.HeartCheck()

		if thing.ConnectedChanged() {
			converter := s.converterLoraCache[thing.GetConverter().GetSN()]

			if thing.IsConnected() {

				s.dataChan <- &model.SpMessage{
					Topic:   fmt.Sprintf("spBv1.0/devices/DBIRTH/%v/%v", s.nodeId, thing.GetId()),
					Payload: thing.DBirth(),
				}

				s.dataChan <- &model.SpMessage{
					Topic:   fmt.Sprintf("spBv1.0/converters/DBIRTH/%v/%v", s.nodeId, converter.sn),
					Payload: loraConverterBirthPayload(converter.converterType),
				}
			} else {
				s.dataChan <- &model.SpMessage{
					Topic:   fmt.Sprintf("spBv1.0/devices/DBIRTH/%v/%v", s.nodeId, thing.GetId()),
					Payload: model.NewPayload(),
				}

				s.dataChan <- &model.SpMessage{
					Topic:   fmt.Sprintf("spBv1.0/converters/DDEATH/%v/%v", s.nodeId, converter.sn),
					Payload: model.NewPayload(),
				}
			}
		}

	}
}

func (s *loraThings) heartBeatRequest() {

	for _, thing := range s.things {
		if c, ok := thing.GetConverter().(*model.LoraConverter); ok {
			c.HeartRequest()
		}
		time.Sleep(3 * time.Second)

	}

}

func (s *loraThings) Request(guid string, data []byte) {

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

func getLoraCmd(t int) byte {
	cmd := 0x00

	// 4: io, 5: 485
	switch t {
	case 4:
		cmd = 0x04
	case 5:
		cmd = 0x05

	}

	return byte(cmd)
}

func (s *loraThings) addConverter(sn string, t int) {

	item := &loraConverter{
		sn:            sn,
		converterType: t,
	}
	s.converterLoraCache[sn] = item
}

func (s *loraThings) Paring(t int) {

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

			// press button data
			case 0x82:
				if len == 9 {

					if panel, found := s.panels[sn]; found {
						key := frame[7]
						actions := panel.Press(key)
						for _, action := range actions {
							s.ActionChan <- action
						}
					}
				}

			// 0x85: 485, 0x83: io
			case 0x85, 0x83:
				if len > 7 {
					if thing, found := s.things[sn]; found {
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

					panel := service.DbPanel{SN: sn}
					if _, err := service.DbAddPanel(&panel); err == nil {

						s.converterPanelCache[sn] = &loraPanel{sn: sn}

						// panel birth

						panelMetric := &model.Payload_Metric{
							Name:     proto.String("panel type"),
							Datatype: proto.Uint32(uint32(model.DataType_Int16)),
							Value:    &model.Payload_Metric_IntValue{IntValue: 3},

							Timestamp: proto.Uint64(uint64(time.Now().UnixMicro())),
						}

						p := model.NewPayload()
						p.Metrics = append(p.Metrics, panelMetric)

						msg := &model.SpMessage{
							Topic:   fmt.Sprintf("spBv1.0/lora-panels/DBIRTH/%v/%v", s.nodeId, sn),
							Payload: p,
						}

						s.dataChan <- msg
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
					if _, err := service.DbAddConverter(converter); err == nil {

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
	if converters, err := service.DbGetConverters("lora"); err == nil {
		for _, c := range converters {
			s.addConverter(c.SN, int(c.ConverterType))

			if c.Guid != nil && c.DeviceType != nil {
				s.addLoraThing(*c.Guid, int(*c.DeviceType), c.SN, int(c.ConverterType))
			}
		}

	}

	// load panel
	if panels, err := service.DbGetPanels(); err == nil {
		for _, p := range panels {
			s.converterPanelCache[p.SN] = &loraPanel{sn: p.SN}

		}
	}

}

func (s *loraThings) updatePanels(data []byte) error {
	p := &panels{
		Panels: make([]*model.LoraPanel, 0, 64),
	}
	err := json.Unmarshal(data, &p)
	if err != nil || p == nil {
		return fmt.Errorf("json unmarshal error")
	}

	s.panels = make(map[string]*model.LoraPanel, len(p.Panels))
	for _, panel := range p.Panels {
		s.panels[panel.SN] = panel
	}

	return nil
}

// func (s *loraThings) has(guid string) bool {

// 	for _, thing := range s.things {
// 		if thing.GetId() == guid {
// 			return true
// 		}
// 	}

// 	return false
// }
