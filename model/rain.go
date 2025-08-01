package model

import (
	"edge/utils"
)

type RainSensor struct {
	raining bool

	guid string

	observer Observer

	Converter

	PassiveReporting *PassiveReporting
	IHeart
}

func NewRainSensor(guid string, c Converter, o Observer) *RainSensor {

	item := &RainSensor{

		guid:      guid,
		Converter: c,

		PassiveReporting: &PassiveReporting{
			Interval: 60 * 5,
		},

		IHeart: new(Heart),

		observer: o,
	}

	return item
}

func (i *RainSensor) Request(command string, params interface{}) {

	var data []byte

	switch command {

	case "getStatus":
		data = []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x01}
	case "setInterval":
		i.PassiveReporting.SetInterval(params)

		return
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

func (i *RainSensor) Response(data []byte) {
	if len(data) < 7 {
		return
	}

	raining := i.raining
	if data[4] == 0x00 {
		i.raining = false
	}

	if data[4] == 0x01 {
		i.raining = true
	}

	changed := (raining != i.raining)

	if changed {

		i.notifyAll()
	}

}

func (i *RainSensor) GetId() string {
	return i.guid
}
func (i *RainSensor) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_RAIN
}

func (i *RainSensor) notifyAll() {

	state := map[string]interface{}{
		"raining": i.raining,
	}
	i.observer.Update(i.guid, state)

}

func (i *RainSensor) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 4800, 0, 8, 1
}

func (i *RainSensor) StartLoopRequest() {
	i.PassiveReporting.StartLoopRequest(i.Request, "getStatus")
}

func (i *RainSensor) StopLoopRequest() {
	i.PassiveReporting.StopLoopRequest()
}

func (i *RainSensor) HeartCheck() {
	i.IHeart.HeartCheck()
	if i.IHeart.IsConnected() && i.IHeart.ConnectedChanged() {
		i.Request("getStatus", nil)
	}
}
