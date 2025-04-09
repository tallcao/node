package model

import (
	"encoding/json"

	"google.golang.org/protobuf/proto"
)

type CommandCreateDevice struct {
	Connection  string `json:"connection"`
	DeviceGUID  string `json:"guid"`
	DeviceType  int64  `json:"type"`
	ConverterSN string `json:"ConverterSN,omitempty"`
	Addr        int    `json:"addr,omitempty"`
}

type CommandResponse struct {
	CommandID string `json:"commandId"`
	Error     string `json:"error"`
	Code      int    `json:"code"`

	Timestamp uint64 `json:"timestamp"`
}

func (cmd CommandResponse) ToPayload() (*Payload, error) {

	payload := &Payload{
		Timestamp: proto.Uint64(0),
		Seq:       proto.Uint64(0),
	}

	v, err := json.Marshal(cmd)

	if err != nil {
		return nil, err
	}

	metric := &Payload_Metric{
		Name:      proto.String("Command Response"),
		Datatype:  proto.Uint32(uint32(DataType_Bytes)),
		Value:     &Payload_Metric_BytesValue{BytesValue: v},
		Timestamp: proto.Uint64(cmd.Timestamp),
	}

	payload.Metrics = append(payload.Metrics, metric)

	return payload, nil

}

type SpMessage struct {
	Topic   string
	Payload *Payload
}

type SpSTATE struct {
	Timestamp int64 `json:"timestamp"`
	Online    bool  `json:"online"`
}

func NewPayload() *Payload {
	return &Payload{
		Timestamp: proto.Uint64(0),
		Seq:       proto.Uint64(0),
	}
}

func NewConverterTypeMetric(v uint32, ts uint64) *Payload_Metric {
	metric := &Payload_Metric{
		Name:      proto.String("converter type"),
		Datatype:  proto.Uint32(uint32(DataType_UInt8)),
		Value:     &Payload_Metric_IntValue{IntValue: v},
		Timestamp: proto.Uint64(ts),
	}

	return metric
}

func NewBdSeqMetric(v byte, ts uint64) *Payload_Metric {
	metric := &Payload_Metric{
		Name:     proto.String("bdSeq"),
		Datatype: proto.Uint32(uint32(DataType_UInt8)),
		Value:    &Payload_Metric_IntValue{IntValue: uint32(v)},

		Timestamp: proto.Uint64(ts),
	}

	return metric
}

type SparkplugDevice interface {
	// DData() *Payload
	DBirth() *Payload
}
