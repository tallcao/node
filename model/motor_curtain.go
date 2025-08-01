package model

import (
	"edge/utils"
	"encoding/json"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MotorCurtain struct {
	percent uint8

	guid string

	observer Observer

	Converter
	addr uint8

	IHeart
}

func NewMotorCurtain(guid string, c Converter, o Observer) *MotorCurtain {

	item := &MotorCurtain{

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

	old := i.percent
	if data[3] == 0x01 && data[4] == 0x10 {

		percent := data[5]

		i.percent = percent
	}

	change := (old != i.percent)
	if change {
		i.notifyAll()
	}
}

func (i *MotorCurtain) GetId() string {
	return i.guid
}
func (i *MotorCurtain) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_MOTOR_CURTAIN
}

func (i *MotorCurtain) notifyAll() {
	state := map[string]interface{}{
		"percent": i.percent,
	}
	i.observer.Update(i.guid, state)
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

func (i *MotorCurtain) UpdateDelta(c mqtt.Client, m mqtt.Message) {

	var desired struct {
		Percent uint8 `json:"percent"`
	}

	err := json.Unmarshal(m.Payload(), &desired)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal motor curtain update delta: %v", err)
		return
	}

	if desired.Percent > 100 {
		log.Printf("ERROR: Invalid percent value %d for motor curtain %s", desired.Percent, i.guid)
		return
	}
	if desired.Percent != i.percent {

		data := []byte{i.addr, 0xFE, 0xFE, 0x03, 0x04, byte(desired.Percent)}
		crc, err := utils.CRC16(data)
		if err != nil {
			log.Printf("ERROR: Failed to calculate CRC16 for motor curtain %s: %v", i.guid, err)
			return
		}
		data = append(data, crc...)
		i.SendFrame(data)
	}
}

func (i *MotorCurtain) CommandRequest(c mqtt.Client, m mqtt.Message) {
	var cmd CommandData

	err := json.Unmarshal(m.Payload(), &cmd)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal motor curtain command: %v", err)
		return
	}

	i.Request(cmd.Command, cmd.Data)
}
