package model

type Am6108 struct {
	co2         uint16
	hcho        uint16
	pm25        uint16
	temperature float32
	humidity    uint16

	guid string

	Converter
	observer Observer

	IHeart
}

func NewAm6108(guid string, c Converter, o Observer) *Am6108 {

	item := &Am6108{

		guid:      guid,
		Converter: c,

		IHeart:   new(Heart),
		observer: o,
	}

	return item
}

func (d *Am6108) Response(data []byte) {
	if len(data) != 25 {
		return
	}

	d.co2 = 256*uint16(data[9]) + uint16(data[10])

	d.hcho = 256*uint16(data[14]) + uint16(data[15])

	d.pm25 = 256*uint16(data[7]) + uint16(data[8])

	d.temperature = (256*float32(data[11]) + float32(data[12]) - 100) / 10.0

	d.humidity = uint16(data[13])

	d.notifyAll()
}

func (d *Am6108) Request(string, interface{}) {

}
func (d *Am6108) GetId() string {
	return d.guid
}
func (i *Am6108) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_AM6108
}

func (i *Am6108) notifyAll() {

	state := map[string]any{
		"co2":         i.co2,
		"hcho":        i.hcho,
		"pm25":        i.pm25,
		"temperature": i.temperature,
		"humidity":    i.humidity,
	}
	i.observer.Update(i.guid, state)

}

func (i *Am6108) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 9600, 0, 8, 1
}
