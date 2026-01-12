package model

import (
	"edge/utils"
	"encoding/binary"
)

// 电表 只显示电量

type ElectricMeterN struct {
	energy float64

	guid string

	addr     byte
	observer Observer

	Converter

	IHeart
}

func NewElectricMeterN(guid string, c Converter, o Observer) *ElectricMeterN {

	item := &ElectricMeterN{

		guid: guid,

		Converter: c,

		IHeart:   new(Heart),
		observer: o,
	}

	if adapter, ok := c.(AddrAdapter); ok {
		item.addr = adapter.GetAddr()
	} else {
		item.addr = 0x01
	}

	return item
}

func (i *ElectricMeterN) Request(command string, params interface{}) {

	data := []byte{i.addr}

	switch command {
	case "heartBeat":
		data = append(data, 0x03, 0x00, 0x63, 0x00, 0x02)
	case "getEnergy":
		data = append(data, 0x03, 0x00, 0x63, 0x00, 0x02)

	default:
		return

	}
	crc, err := utils.CRC16(data)
	if err != nil {
		return
	}
	data = append(data, crc...)

	i.SendFrame(data)

}

func (d *ElectricMeterN) Response(data []byte) {

	if len(data) < 7 {
		return
	}

	if len(data) == 9 && data[1] == 0x03 {

		intValue := int64(binary.BigEndian.Uint32(data[3:7]))

		d.energy = float64(intValue) / 100

		d.notifyEnergy()

	}

}

func (i *ElectricMeterN) GetId() string {
	return i.guid
}
func (i *ElectricMeterN) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_ELECTRIC_METER_N
}

func (i *ElectricMeterN) notifyEnergy() {

	state := map[string]any{
		"energy": i.energy,
	}

	i.observer.Update(i.guid, state)

}

func (i *ElectricMeterN) HeartCheck() {
	i.IHeart.HeartCheck()
	if i.IHeart.IsConnected() && i.IHeart.ConnectedChanged() {
		i.Request("getEnergy", nil)
	}
}

func (i *ElectricMeterN) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 9600, 0, 8, 1
}
