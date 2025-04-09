package view

import (
	"edge/model"
	"fmt"
)

type SparkPlugView struct {
	id   string
	node string

	dataCh chan<- *model.SpMessage
}

func NewSparkPlugView(id string, node string, ch chan<- *model.SpMessage) *SparkPlugView {

	return &SparkPlugView{
		id:     id,
		node:   node,
		dataCh: ch,
	}
}

func (v *SparkPlugView) Update(data *model.Payload) {

	msg := &model.SpMessage{
		Topic:   fmt.Sprintf("spBv1.0/devices/DDATA/%v/%v", v.node, v.id),
		Payload: data,
	}
	v.dataCh <- msg
}
func (v *SparkPlugView) GetID() string {
	return v.id
}
