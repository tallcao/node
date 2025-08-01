package model

type SoilSensor struct {
	temperature  float32
	humidity     float32
	conductivity float32
	ph           float32

	guid string

	observer Observer

	Converter

	PassiveReporting *PassiveReporting

	IHeart
}

func NewSoilSensor(guid string, c Converter, o Observer) *SoilSensor {

	item := &SoilSensor{

		guid:      guid,
		Converter: c,

		PassiveReporting: &PassiveReporting{
			Interval: 60 * 10,
		},

		IHeart:   new(Heart),
		observer: o,
	}

	return item
}

func (i *SoilSensor) Request(command string, params interface{}) {

	switch command {
	case "getStatus":
		data := []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x04, 0x44, 0x09}
		i.SendFrame(data)
	case "setInterval":
		i.PassiveReporting.SetInterval(params)
		// i.notifyAll()
	default:
		return
	}
}

func (i *SoilSensor) Response(data []byte) {
	if len(data) != 13 {
		return
	}

	if data[1] != 0x03 {
		return
	}

	// var humidity float32
	// var temperature float32
	// var conductivity float32
	// var ph float32

	i.humidity = (float32(data[3])*256 + float32(data[4])) / 10

	flag := data[5] & 0x80

	if flag == 0 {
		i.temperature = (float32(data[5])*256 + float32(data[6])) / 10
	} else {
		i.temperature = (float32(data[5]^0xff)*256 + float32(data[6]^0xff) + 1) / 10
	}

	i.conductivity = float32(data[7])*256 + float32(data[8])
	i.ph = (float32(data[9])*256 + float32(data[10])) / 10

	i.notifyAll()
}

func (i *SoilSensor) GetId() string {
	return i.guid
}

func (i *SoilSensor) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_SOIL_SENSOR
}

func (i *SoilSensor) notifyAll() {

	state := map[string]interface{}{
		"temperature":  i.temperature,
		"humidity":     i.humidity,
		"conductivity": i.conductivity,
		"ph":           i.ph,
	}
	i.observer.Update(i.guid, state)

}

func (i *SoilSensor) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 4800, 0, 8, 1
}

func (i *SoilSensor) StartLoopRequest() {
	i.PassiveReporting.StartLoopRequest(i.Request, "getStatus")
}

func (i *SoilSensor) StopLoopRequest() {
	i.PassiveReporting.StopLoopRequest()
}

func (i *SoilSensor) HeartCheck() {
	i.IHeart.HeartCheck()
	if i.IHeart.IsConnected() && i.IHeart.ConnectedChanged() {
		i.Request("getStatus", nil)
	}
}
