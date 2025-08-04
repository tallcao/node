package model

import (
	"edge/utils"
	"encoding/json"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Motor struct {
	percent   uint8
	direction uint8
	pull      uint8
	status    uint8

	guid string

	observer Observer

	Converter
	addr uint8

	IHeart
}

type percentData struct {
	Data int `json:"data"`
}

func NewMotor(guid string, c Converter, o Observer) *Motor {

	item := &Motor{

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

	i.SendFrame(result)
}

func (i *Motor) Response(data []byte) {
	if len(data) < 7 {
		return
	}

	if data[0] != 0x55 {
		return

	}

	// percent :=i.percent
	// direction := i.direction
	// pull := i.pull
	// status := i.status

	if data[3] == 0x03 && data[4] == 0x03 && len(data) == 16 {
		i.percent = data[13]
	}

	if data[3] == 0x01 && len(data) == 9 {

		addr := data[4]
		val := data[6]
		if addr == 0x02 {
			i.percent = val

		}
		if addr == 0x03 {
			i.direction = val

		}
		if addr == 0x04 {
			i.pull = val

		}
		if addr == 0x05 {
			i.status = val

		}
	}

	i.notifyAll()

}

func (i *Motor) GetId() string {
	return i.guid
}

func (i *Motor) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_MOTOR
}

func (i *Motor) notifyAll() {

	state := map[string]interface{}{
		"percent":   i.percent,
		"direction": i.direction,
		"pull":      i.pull,
		"status":    i.status,
	}
	i.observer.Update(i.guid, state)

}

func (i *Motor) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 9600, 0, 8, 1
}

func (i *Motor) HeartCheck() {
	i.IHeart.HeartCheck()
	if i.IHeart.IsConnected() && i.IHeart.ConnectedChanged() {
		i.Request("getPercent", nil)
	}
}

func (i *Motor) GetAccepted(c mqtt.Client, m mqtt.Message) {
	var desired struct {
		Percent uint8 `json:"percent"`
	}

	err := json.Unmarshal(m.Payload(), &desired)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal e-valve update delta: %v", err)
		return
	}

	if desired.Percent > 100 {
		log.Printf("ERROR: Invalid percent value %d for motor curtain %s", desired.Percent, i.guid)
		return
	}

	data := []byte{i.addr, 0xFE, 0xFE, 0x03, 0x04, byte(desired.Percent)}
	crc, err := utils.CRC16(data)
	if err != nil {
		log.Printf("ERROR: Failed to calculate CRC16 for motor curtain %s: %v", i.guid, err)
		return
	}
	data = append(data, crc...)
	i.SendFrame(data)
}

func (i *Motor) UpdateDelta(c mqtt.Client, m mqtt.Message) {

	var desired struct {
		Percent uint8 `json:"percent"`
	}

	err := json.Unmarshal(m.Payload(), &desired)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal e-valve update delta: %v", err)
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

func (i *Motor) CommandRequest(c mqtt.Client, m mqtt.Message) {
	var cmd CommandData

	err := json.Unmarshal(m.Payload(), &cmd)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal motor command: %v", err)
		return
	}

	i.Request(cmd.Command, cmd.Data)
}
