package model

import (
	"edge/utils"
	"encoding/json"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MotorFR struct {

	// 0:stop, 1: open, 2: close

	status uint8

	guid string

	observer Observer

	Converter
	addr uint8

	IHeart
}

func NewMotorFR(guid string, c Converter, o Observer) *MotorFR {

	item := &MotorFR{

		guid:      guid,
		Converter: c,

		IHeart: new(Heart),

		observer: o,
	}

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
	i.SendFrame(result)

}

func (i *MotorFR) Response(data []byte) {

	n := len(data)
	if n == 0 {
		return
	}

	status := i.status

	if n == 8 {

		if data[1] == 0x06 {
			switch data[5] {
			case 0x01:
				i.status = 1
			case 0x02:
				i.status = 2
			case 0x00:
				i.status = 0

			}

		}
	}

	changed := (status != i.status)

	if changed {
		i.notifyAll()
	}
}

func (i *MotorFR) GetId() string {
	return i.guid
}
func (i *MotorFR) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_MOTOR_FR
}

func (i *MotorFR) notifyAll() {

	state := map[string]interface{}{
		"status": i.status,
	}
	i.observer.Update(i.guid, state)

}

func (i *MotorFR) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 9600, 0, 8, 1
}
func (i *MotorFR) GetAccepted(c mqtt.Client, m mqtt.Message) {
	var desired struct {
		Status uint8 `json:"status"`
	}

	err := json.Unmarshal(m.Payload(), &desired)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal e-valve update delta: %v", err)
		return
	}

	data := []byte{i.addr, 0x06, 0x00, 0x00, 0x00, byte(desired.Status)}

	crc, err := utils.CRC16(data)
	if err != nil {
		log.Printf("ERROR: Failed to calculate CRC16 for motor curtain %s: %v", i.guid, err)
		return
	}
	data = append(data, crc...)
	i.SendFrame(data)
}

func (i *MotorFR) UpdateDelta(c mqtt.Client, m mqtt.Message) {

	var desired struct {
		Status uint8 `json:"status"`
	}

	err := json.Unmarshal(m.Payload(), &desired)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal e-valve update delta: %v", err)
		return
	}

	if desired.Status != i.status {

		data := []byte{i.addr, 0x06, 0x00, 0x00, 0x00, byte(desired.Status)}

		crc, err := utils.CRC16(data)
		if err != nil {
			log.Printf("ERROR: Failed to calculate CRC16 for motor curtain %s: %v", i.guid, err)
			return
		}
		data = append(data, crc...)
		i.SendFrame(data)
	}
}

func (i *MotorFR) CommandRequest(c mqtt.Client, m mqtt.Message) {
	var cmd CommandData

	err := json.Unmarshal(m.Payload(), &cmd)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal motor-fr command: %v", err)
		return
	}

	i.Request(cmd.Command, cmd.Data)
}
