package model

import (
	"edge/utils"
	"encoding/json"
)

type MotorCurtain struct {
	// todo
	Percent byte `json:"percent"`

	guid string

	observer Observer

	converter Converter
	addr      uint8

	heart *Heart
}

func NewMotorCurtain(guid string, c Converter, o Observer) *MotorCurtain {

	item := &MotorCurtain{
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

	i.converter.SendFrame(result)
}

func (i *MotorCurtain) Response(data []byte) {

	if len(data) != 23 {
		return
	}

	if data[3] == 0x01 && data[4] == 0x10 {
		i.Percent = data[5]

	}

	// i.notifyAll()

}

func (i *MotorCurtain) GetId() string {
	return i.guid
}
func (i *MotorCurtain) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_MOTOR_CURTAIN
}
func (i *MotorCurtain) GetConverter() Converter {
	return i.converter
}

func (i *MotorCurtain) notifyAll() {

	// for _, observer := range i.observerList {
	// 	observer.Update()
	// }
}

func (i *MotorCurtain) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 9600, 0, 8, 1
}

func (i *MotorCurtain) HeartBeat() {
	i.heart.HeartBeat()
}

func (i *MotorCurtain) HeartCheck() {
	i.heart.HeartCheck()
	if i.heart.Conected && i.heart.Changed() {
		i.Request("getPercent", nil)
	}
}

func (i *MotorCurtain) IsConnected() bool {
	return i.heart.Conected
}

func (i *MotorCurtain) ConnectedChanged() bool {
	return i.heart.Changed()
}
