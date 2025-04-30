package view

import (
	"edge/model"
	"fmt"
)

type SparkPlugView struct {
	node string

	dataCh chan<- *model.SpMessage
}

func NewSparkPlugView(node string, ch chan<- *model.SpMessage) *SparkPlugView {

	return &SparkPlugView{
		node:   node,
		dataCh: ch,
	}
}

func (v *SparkPlugView) Update(id string, data *model.Payload) {

	msg := &model.SpMessage{
		Topic:   fmt.Sprintf("spBv1.0/devices/DDATA/%v/%v", v.node, id),
		Payload: data,
	}
	v.dataCh <- msg
}
