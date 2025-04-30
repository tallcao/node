package model

import "google.golang.org/protobuf/proto"

type LightModuleChild struct {
	guid   string
	no     int
	status *Payload_Metric

	observer Observer
}

func NewLightModuleChild(guid string, no int, o Observer) *LightModuleChild {

	return &LightModuleChild{

		guid: guid,
		no:   no,

		status: &Payload_Metric{
			Name:      proto.String("on"),
			Datatype:  proto.Uint32(uint32(DataType_Boolean)),
			Timestamp: proto.Uint64(0),
		},

		observer: o,
	}

}

func (i *LightModuleChild) Set(new bool) {

	old := i.status.GetBooleanValue()
	i.status.Value = &Payload_Metric_BooleanValue{new}

	changed := new != old
	if changed {
		i.notifyAll()
	}
}

func (i *LightModuleChild) GetId() string {
	return i.guid
}

func (i *LightModuleChild) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.status)

	return p

}

func (i *LightModuleChild) notifyAll() {

	p := NewPayload()

	p.Metrics = append(p.Metrics, i.status)

	i.observer.Update(i.guid, p)

}
