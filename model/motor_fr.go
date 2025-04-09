package model

import (
	"edge/utils"
	"time"

	"google.golang.org/protobuf/proto"
)

type MotorFR struct {

	// 0:stop, 1: close, 2: open

	status *Payload_Metric

	guid string

	observerList []Observer

	converter Converter
	addr      uint8

	heart *Heart
}

func NewMotorFR(guid string, c Converter, o Observer) *MotorFR {

	item := &MotorFR{
		status: &Payload_Metric{
			Name:      proto.String("status"),
			Datatype:  proto.Uint32(uint32(DataType_UInt8)),
			Timestamp: proto.Uint64(0),
		},

		guid:      guid,
		converter: c,

		heart: new(Heart),
	}

	item.register(o)
	if adapter, ok := c.(AddrAdapter); ok {
		item.addr = adapter.GetAddr()
	} else {
		item.addr = 0x01
	}
	return item
}

func (i *MotorFR) Request(command string, params interface{}) {

	result := make([]byte, 0, 8)

	result = append(result, i.addr, 0x06, 0x00, 0x00, 0x00)

	switch command {
	case "heartBeat":
		result = []byte{i.addr, 0x03, 0x00, 0x00, 0x00, 0x01}
	case "stop":
		result = append(result, 0x00)

	case "close":
		result = append(result, 0x02)

	case "open":
		result = append(result, 0x01)

	default:
		return
	}

	crc, err := utils.CRC16(result)
	if err != nil {
		return
	}
	result = append(result, crc...)
	i.converter.SendFrame(result)

}

func (i *MotorFR) Response(data []byte) {

	n := len(data)
	if n == 0 {

		return
	}

	status := i.status.GetIntValue()

	if n == 8 {

		if data[1] == 0x06 {
			switch data[5] {
			case 0x01:
				status = 2
			case 0x02:
				status = 1
			case 0x00:
				status = 0

			}

			changed := (status != i.status.GetIntValue())

			i.status.Value = &Payload_Metric_IntValue{status}
			*i.status.Timestamp = uint64(time.Now().UnixMicro())

			if changed {
				i.notifyAll()

			}
		}
	}

}

func (i *MotorFR) GetId() string {
	return i.guid
}
func (i *MotorFR) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_MOTOR_FR
}

func (i *MotorFR) GetConverter() Converter {
	return i.converter
}

func (i *MotorFR) register(o Observer) {
	i.observerList = append(i.observerList, o)
}

func (i *MotorFR) deregister(o Observer) {
	i.observerList = removeFromslice(i.observerList, o)
}

func (i *MotorFR) notifyAll() {
	p := NewPayload()
	p.Metrics = append(p.Metrics, i.status)

	for _, observer := range i.observerList {
		observer.Update(p)
	}

}

func (i *MotorFR) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 9600, 0, 8, 1
}

func (i *MotorFR) HeartBeat() {
	i.heart.HeartBeat()
}

func (i *MotorFR) HeartCheck() {
	i.heart.HeartCheck()
}

func (i *MotorFR) IsConnected() bool {
	return i.heart.Conected
}

func (i *MotorFR) ConnectedChanged() bool {
	return i.heart.Changed()
}

func (i *MotorFR) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.status)

	return p

}
