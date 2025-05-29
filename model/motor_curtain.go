package model

import (
	"edge/utils"
	"encoding/json"
	"time"

	"google.golang.org/protobuf/proto"
)

type MotorCurtain struct {
	percent *Payload_Metric

	guid string

	observer Observer

	Converter
	addr uint8

	IHeart
}

func NewMotorCurtain(guid string, c Converter, o Observer) *MotorCurtain {

	item := &MotorCurtain{
		percent: &Payload_Metric{
			Name:      proto.String("percent"),
			Datatype:  proto.Uint32(uint32(DataType_UInt8)),
			Timestamp: proto.Uint64(0),
		},

		guid:      guid,
		Converter: c,

		IHeart: new(Heart),

		observer: o,
	}

	if adapter, ok := c.(AddrAdapter); ok {
		item.addr = adapter.GetAddr()
	} else {
		item.addr = 0x55
	}
	return item
}

func (i *MotorCurtain) Request(command string, params interface{}) {

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
	case "getPercent":
		result = append(result, 0x01, 0x02, 0x01)

	case "heartBeat":
		result = append(result, 0x01, 0x02, 0x01)
	}

	crc, err := utils.CRC16(result)
	if err != nil {
		return
	}
	result = append(result, crc...)

	i.SendFrame(result)
}

func (i *MotorCurtain) Response(data []byte) {

	if len(data) != 23 {
		return
	}

	if data[3] == 0x01 && data[4] == 0x10 {

		percent := data[5]
		ts := uint64(time.Now().UnixMicro())

		i.percent.Value = &Payload_Metric_IntValue{uint32(percent)}
		*i.percent.Timestamp = ts

	}

	i.notifyAll()

}

func (i *MotorCurtain) GetId() string {
	return i.guid
}
func (i *MotorCurtain) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_MOTOR_CURTAIN
}

func (i *MotorCurtain) notifyAll() {

	p := NewPayload()
	p.Metrics = append(p.Metrics, i.percent)
	i.observer.Update(i.guid, p)
}

func (i *MotorCurtain) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 9600, 0, 8, 1
}

func (i *MotorCurtain) HeartCheck() {
	i.IHeart.HeartCheck()
	if i.IHeart.IsConnected() && i.IHeart.ConnectedChanged() {
		i.Request("getPercent", nil)
	}
}

func (i *MotorCurtain) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.percent)

	return p

}
