package model

type BodySensorV4 struct {
	body bool
	guid string

	observer Observer

	Converter

	IHeart
}

func NewBodySensorV4(guid string, c Converter, o Observer) *BodySensorV4 {

	item := &BodySensorV4{

		guid:      guid,
		Converter: c,

		IHeart:   new(Heart),
		observer: o,
	}

	return item
}

func (i *BodySensorV4) Request(command string, params interface{}) {

}

func (i *BodySensorV4) Response(data []byte) {

	if len(data) < 8 {
		return
	}
	body := i.body

	if data[6] == 0xAA {
		i.body = true
	}

	if data[6] == 0x55 {
		i.body = false
	}

	changed := (body != i.body)

	if changed {

		i.notifyAll()
	}

}

func (i *BodySensorV4) GetId() string {
	return i.guid
}

func (i *BodySensorV4) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_BODY_V4
}

func (i *BodySensorV4) notifyAll() {

	state := map[string]any{
		"body": i.body,
	}
	i.observer.Update(i.guid, state)

}

func (i *BodySensorV4) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 9600, 0, 8, 1
}
