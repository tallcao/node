package model

import (
	"edge/utils"
	"encoding/json"
	"time"

	"google.golang.org/protobuf/proto"
)

type Motor struct {
	percent   *Payload_Metric
	direction *Payload_Metric
	pull      *Payload_Metric
	status    *Payload_Metric

	guid string

	observer Observer

	converter Converter
	addr      uint8

	heart *Heart
}

type percentData struct {
	Data int `json:"data"`
}

func NewMotor(guid string, c Converter, o Observer) *Motor {

	item := &Motor{
		percent: &Payload_Metric{
			Name:      proto.String("percent"),
			Datatype:  proto.Uint32(uint32(DataType_UInt8)),
			Timestamp: proto.Uint64(0),
		},
		direction: &Payload_Metric{
			Name:      proto.String("direction"),
			Datatype:  proto.Uint32(uint32(DataType_UInt8)),
			Timestamp: proto.Uint64(0),
		},
		pull: &Payload_Metric{
			Name:      proto.String("pull"),
			Datatype:  proto.Uint32(uint32(DataType_UInt8)),
			Timestamp: proto.Uint64(0),
		},
		status: &Payload_Metric{
			Name:      proto.String("status"),
			Datatype:  proto.Uint32(uint32(DataType_UInt8)),
			Timestamp: proto.Uint64(0),
		},

		guid:      guid,
		converter: c,

		heart: new(Heart),

		observer: o,
	}

	if adapter, ok := c.(AddrAdapter); ok {
		item.addr = adapter.GetAddr()
	} else {
		item.addr = 0x55
	}

	return item
}

func (i *Motor) Request(command string, params interface{}) {
	result := make([]byte, 0, 16)

	result = append(result, i.addr, 0xFE, 0xFE)

	switch command {
	case "open":
		result = append(result, 0x03, 0x01)
	case "close":
		result = append(result, 0x03, 0x02)
	case "stop":
		result = append(result, 0x03, 0x03)
	case "percent":
		result = append(result, 0x03, 0x04)
		if jsonData, err := json.Marshal(params); err == nil {
			p := &percentData{}
			if err := json.Unmarshal(jsonData, &p); err == nil && p != nil {
				result = append(result, byte(p.Data))
			}
		}

	case "setUp":
		result = append(result, 0x03, 0x05, 0x01)
	case "setDown":
		result = append(result, 0x03, 0x05, 0x02)
	case "getPercent":
		result = append(result, 0x01, 0x02, 0x01)
	}

	crc, err := utils.CRC16(result)
	if err != nil {
		return
	}
	result = append(result, crc...)

	i.converter.SendFrame(result)
}

func (i *Motor) Response(data []byte) {
	if len(data) < 7 {
		return
	}

	if data[0] != 0x55 {
		return

	}

	var percent uint8
	var direction uint8
	var pull uint8
	var status uint8

	if data[3] == 0x03 && data[4] == 0x03 && len(data) == 16 {
		percent = data[13]
	}

	if data[3] == 0x01 && len(data) == 9 {

		addr := data[4]
		val := data[6]
		if addr == 0x02 {
			percent = val

		}
		if addr == 0x03 {
			direction = val

		}
		if addr == 0x04 {
			pull = val

		}
		if addr == 0x05 {
			status = val

		}
	}

	ts := uint64(time.Now().UnixMicro())

	i.percent.Value = &Payload_Metric_IntValue{uint32(percent)}
	*i.percent.Timestamp = ts

	i.direction.Value = &Payload_Metric_IntValue{uint32(direction)}
	*i.direction.Timestamp = ts

	i.pull.Value = &Payload_Metric_IntValue{uint32(pull)}
	*i.pull.Timestamp = ts

	i.status.Value = &Payload_Metric_IntValue{uint32(status)}
	*i.status.Timestamp = ts

	i.notifyAll()

}

func (i *Motor) GetId() string {
	return i.guid
}

func (i *Motor) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_MOTOR
}

func (i *Motor) GetConverter() Converter {
	return i.converter
}

func (i *Motor) notifyAll() {

	p := NewPayload()
	p.Metrics = append(p.Metrics, i.percent, i.direction, i.pull, i.status)
	i.observer.Update(i.guid, p)

}

func (i *Motor) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 9600, 0, 8, 1
}

func (i *Motor) HeartBeat() {
	i.heart.HeartBeat()
}

func (i *Motor) HeartCheck() {
	i.heart.HeartCheck()
	if i.heart.Conected && i.heart.Changed() {
		i.Request("getPercent", nil)
	}
}

func (i *Motor) IsConnected() bool {
	return i.heart.Conected
}

func (i *Motor) ConnectedChanged() bool {
	return i.heart.Changed()
}

func (i *Motor) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.percent, i.direction, i.pull, i.status)

	return p

}
