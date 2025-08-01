package model

type BodySensor struct {
	body bool

	guid string

	observer Observer

	Converter

	IHeart
}

func NewBodySensor(guid string, c Converter, o Observer) *BodySensor {

	item := &BodySensor{

		guid:      guid,
		Converter: c,

		IHeart: new(Heart),

		observer: o,
	}

	return item
}

func (i *BodySensor) Request(command string, params interface{}) {

}

func (i *BodySensor) Response(data []byte) {

	if len(data) < 7 {
		return
	}
	body := i.body

	if data[4] == 0x00 {
		i.body = false
	}

	if data[4] == 0x01 {
		i.body = true
	}

	changed := (body != i.body)

	if changed {

		i.notifyAll()
	}

}

func (i *BodySensor) GetId() string {
	return i.guid
}

func (i *BodySensor) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_BODY
}

func (i *BodySensor) notifyAll() {
	state := map[string]any{
		"body": i.body,
	}
	i.observer.Update(i.guid, state)

}

func (i *BodySensor) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 9600, 0, 8, 1
}
