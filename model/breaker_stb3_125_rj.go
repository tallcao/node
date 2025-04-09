package model

import (
	"edge/utils"
	"time"

	"google.golang.org/protobuf/proto"
)

// stb3-125/RJ 带计量断路器

type Breaker_STB3_125_RJ struct {
	lock     *Payload_Metric
	on       *Payload_Metric
	quantity *Payload_Metric

	guid string

	addr         byte
	observerList []Observer

	converter Converter

	PassiveReporting *PassiveReporting

	heart *Heart
}

func NewBreaker_STB3_125_RJ(guid string, c Converter, o Observer) *Breaker_STB3_125_RJ {

	item := &Breaker_STB3_125_RJ{
		lock: &Payload_Metric{
			Name:      proto.String("lock"),
			Datatype:  proto.Uint32(uint32(DataType_Boolean)),
			Timestamp: proto.Uint64(0),
		},
		on: &Payload_Metric{
			Name:      proto.String("on"),
			Datatype:  proto.Uint32(uint32(DataType_Boolean)),
			Timestamp: proto.Uint64(0),
		},
		quantity: &Payload_Metric{
			Name:      proto.String("quantity"),
			Datatype:  proto.Uint32(uint32(DataType_Float)),
			Timestamp: proto.Uint64(0),
		},

		guid:      guid,
		converter: c,

		PassiveReporting: &PassiveReporting{
			Interval: 60 * 60,
		},

		heart: new(Heart),
	}

	item.register(o)

	if adapter, ok := c.(AddrAdapter); ok {
		item.addr = adapter.GetAddr()
	} else {
		item.addr = 0x01
	}

	return item
}

func (i *Breaker_STB3_125_RJ) Request(command string, params interface{}) {

	var data []byte
	data = append(data, i.addr)

	switch command {
	case "setInterval":
		i.PassiveReporting.SetInterval(params)
		// i.notifyAll()
		return
	case "heartBeat":
		data = append(data, 0x04, 0x00, 0x00, 0x00, 0x01)
	case "getStatus":
		data = append(data, 0x04, 0x00, 0x00, 0x00, 0x01)
	case "getQuantity":
		data = append(data, 0x04, 0x00, 0x25, 0x00, 0x02)
	case "on":
		data = append(data, 0x05, 0x00, 0x01, 0xFF, 0x00)
	case "off":
		data = append(data, 0x05, 0x00, 0x01, 0x00, 0x00)
	case "toggle":
		v := byte(0xff)
		if i.on.GetBooleanValue() {
			v = 0
		}
		data = append(data, 0x05, 0x00, 0x01, v, 0x00)

	default:
		return
	}
	crc, err := utils.CRC16(data)
	if err != nil {
		return
	}
	data = append(data, crc...)

	i.converter.SendFrame(data)

}

func (d *Breaker_STB3_125_RJ) Response(data []byte) {

	if len(data) < 2 {
		return
	}

	on := d.on.GetBooleanValue()
	lock := d.lock.GetBooleanValue()
	quantity := d.quantity.GetFloatValue()

	if len(data) == 8 && data[1] == 0x05 {

		if data[4] == 0xFF {
			on = true
		}

		if data[4] == 0x00 {
			on = false
		}

	}

	if len(data) == 7 && data[1] == 0x04 {

		if data[3] == 0x00 {
			lock = false

		}
		if data[3] == 0x01 {
			lock = true

		}
		if data[4] == 0xF0 {
			lock = true

		}
		if data[4] == 0x0F {
			lock = false

		}

	}

	if len(data) == 9 && data[1] == 0x04 {
		quantity = float32(data[3])*256 + float32(data[4]) + (float32(data[5])*256+float32(data[6]))*0.001

	}

	onChanged := (on != d.on.GetBooleanValue())
	lockChanged := (lock != d.lock.GetBooleanValue())
	quantityChanged := (quantity != d.quantity.GetFloatValue())

	ts := uint64(time.Now().UnixMicro())

	d.on.Value = &Payload_Metric_BooleanValue{on}
	*d.on.Timestamp = ts

	d.lock.Value = &Payload_Metric_BooleanValue{lock}
	*d.lock.Timestamp = ts

	d.quantity.Value = &Payload_Metric_FloatValue{quantity}
	*d.quantity.Timestamp = ts

	p := NewPayload()

	if onChanged {
		p.Metrics = append(p.Metrics, d.on)
	}

	if lockChanged {
		p.Metrics = append(p.Metrics, d.lock)

	}
	if quantityChanged {
		p.Metrics = append(p.Metrics, d.quantity)

	}

	if len(p.Metrics) > 0 {
		for _, observer := range d.observerList {
			observer.Update(p)
		}
	}

}

func (i *Breaker_STB3_125_RJ) GetId() string {
	return i.guid
}

func (i *Breaker_STB3_125_RJ) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_BREAKER_STB3_125_RJ
}

func (i *Breaker_STB3_125_RJ) GetConverter() Converter {
	return i.converter
}

func (i *Breaker_STB3_125_RJ) register(o Observer) {
	i.observerList = append(i.observerList, o)
}

func (i *Breaker_STB3_125_RJ) deregister(o Observer) {
	i.observerList = removeFromslice(i.observerList, o)
}

func (i *Breaker_STB3_125_RJ) notifyAll() {

	p := NewPayload()
	p.Metrics = append(p.Metrics, i.on, i.lock, i.quantity)

	for _, observer := range i.observerList {
		observer.Update(p)
	}

}

func (i *Breaker_STB3_125_RJ) StartLoopRequest() {
	i.PassiveReporting.StartLoopRequest(i.Request, "getQuantity")
}

func (i *Breaker_STB3_125_RJ) StopLoopRequest() {
	i.PassiveReporting.StopLoopRequest()
}

func (i *Breaker_STB3_125_RJ) HeartBeat() {
	i.heart.HeartBeat()
}

func (i *Breaker_STB3_125_RJ) HeartCheck() {
	i.heart.HeartCheck()
	if i.heart.Conected && i.heart.Changed() {
		i.Request("getStatus", nil)
		time.Sleep(3 * time.Second)
		i.Request("getQuantity", nil)
	}
}

func (i *Breaker_STB3_125_RJ) IsConnected() bool {
	return i.heart.Conected
}

func (i *Breaker_STB3_125_RJ) ConnectedChanged() bool {
	return i.heart.Changed()
}

func (i *Breaker_STB3_125_RJ) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.lock, i.on, i.quantity)

	return p

}
