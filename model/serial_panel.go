package model

import "fmt"

type SerialPanel struct {
	action string

	guid string

	addr byte

	observer Observer

	Converter

	IHeart
}

func NewSerialPanel(guid string, c Converter, o Observer) *SerialPanel {

	item := &SerialPanel{

		guid: guid,

		Converter: c,
		IHeart:    new(Heart),
		observer:  o,
	}

	if adapter, ok := c.(AddrAdapter); ok {
		item.addr = adapter.GetAddr()
	} else {
		item.addr = 0x02
	}

	return item
}

func (i *SerialPanel) Request(command string, params any) {

}

// todo
func (i *SerialPanel) Response(data []byte) {

	if len(data) != 8 {
		return
	}

	cmd := data[1]
	code := data[4]
	value := data[5]

	if cmd != 0x03 || i.addr > 42 {
		return
	}

	keyNum := code - (i.addr-1)*6

	keyValue := ""

	if value > 0 {
		keyValue = "down"
	} else {
		keyValue = "up"

	}

	i.action = fmt.Sprintf("%v_%v", keyValue, keyNum)

	i.notifyAll()

}

func (i *SerialPanel) GetId() string {
	return i.guid
}

func (i *SerialPanel) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_LORA_PANEL
}

func (i *SerialPanel) notifyAll() {

	state := map[string]interface{}{
		"action": i.action,
	}

	i.observer.Update(i.guid, state)

}
