package model

type LoraPanel struct {
	action string

	guid string

	observer Observer

	Converter

	IHeart
}

func NewLoraPanel(guid string, c Converter, o Observer) *LoraPanel {

	item := &LoraPanel{

		guid: guid,

		Converter: c,
		IHeart:    new(Heart),
		observer:  o,
	}

	return item
}

func (i *LoraPanel) Request(command string, params any) {

}

func (i *LoraPanel) Response(data []byte) {

	if len(data) != 1 {
		return
	}

	v := ""

	switch data[0] {
	case 1:
		v = "press_1"
	case 2:
		v = "press_2"
	case 4:
		v = "press_3"
	}

	i.action = v

	i.notifyAll()

}

func (i *LoraPanel) GetId() string {
	return i.guid
}

func (i *LoraPanel) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_LORA_PANEL
}

func (i *LoraPanel) notifyAll() {

	state := map[string]any{
		"action": i.action,
	}

	i.observer.Update(i.guid, state)

}

func (i *LoraPanel) HeartRequest() {

}
