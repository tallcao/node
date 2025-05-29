package core

import (
	"database/sql"
	"edge/model"
	"edge/service"
	"edge/system"
	"edge/utils"
	"encoding/json"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"google.golang.org/protobuf/proto"
)

type nodeConfig struct {
	ID     string `json:"id"`
	Cafile string `json:"cafile"`
	Broker string `json:"broker"`
	Dbfile string `json:"dbfile"`
}

type Node struct {
	id string

	// sparkplug
	primaryHostAppOnline bool

	loraThings   *loraThings
	canThings    *canThings
	serialThings *serialThings

	sparkService *service.SparkplugService

	dataChan chan *model.SpMessage

	seq byte

	db *sql.DB
}

func NewNode(file string, dbus *service.DbusService) *Node {

	content, err := os.ReadFile(file)

	if err != nil {
		return nil
	}

	config := &nodeConfig{}
	err = json.Unmarshal(content, config)
	if err != nil {
		return nil
	}

	id := config.ID
	ca := config.Cafile
	uri := config.Broker

	ch := make(chan *model.SpMessage)

	db, err := sql.Open("sqlite3", config.Dbfile)

	if err != nil {
		return nil
	}

	return &Node{

		id: id,

		seq: getSeq(),

		canThings:    NewCanThings(dbus.CanConnection(), ch, id, db),
		loraThings:   NewLoraThings(dbus.LoraConnection(), ch, id, db),
		serialThings: NewSerialThings(dbus.SerialConnection(), ch, id, db),

		sparkService: service.NewSparkplugService(id, ca, uri),

		dataChan: ch,

		db: db,
	}

}

func (n *Node) Init() {

	n.loraThings.Init()
	n.canThings.Init()
	n.serialThings.Init()
	n.sparkService.SetOnConn(n.onConnectHandler)

}

func (n *Node) stateCallback(c mqtt.Client, m mqtt.Message) {

	state := model.SpSTATE{}
	err := json.Unmarshal(m.Payload(), &state)
	if err != nil {
		return
	}

	n.primaryHostAppOnline = state.Online

}

// todo
func getSeq() byte {

	return 0
}

func (n *Node) createDevice(p *model.Payload) {
	commandID := ""

	data := &model.CommandCreateDevice{}

	for _, m := range p.GetMetrics() {

		if m.GetName() == "Command ID" {
			commandID = m.GetStringValue()
			continue
		}

		if m.GetName() == "Node Control/Create Device" {

			v := m.GetBytesValue()

			err := json.Unmarshal(v, &data)

			if err != nil {
				return
			}

			continue
		}

	}

	if commandID == "" || data == nil {
		return
	}

	thingType := model.DEVICE_TYPE(data.DeviceType)
	guid := data.DeviceGUID

	var response *model.CommandResponse

	switch data.Connection {
	case "can":
		response = n.canThings.Create(guid, thingType, data.ConverterSN)
	case "serial":
		response = n.serialThings.Create(guid, thingType, uint8(data.Addr))
	case "lora":
		response = n.loraThings.Create(guid, thingType, data.ConverterSN)
	default:
		return
	}
	response.Timestamp = uint64(time.Now().UnixMicro())
	response.CommandID = commandID

	if payload, err := response.ToPayload(); err == nil {
		msg := &model.SpMessage{
			Topic:   fmt.Sprintf("spBv1.0/devices/NDATA/%v", n.id),
			Payload: payload,
		}

		n.dataChan <- msg

	}

}

func (n *Node) loraPairing(p *model.Payload) {

	commandID := ""

	duration := 0

	for _, m := range p.GetMetrics() {

		if m.GetName() == "Command ID" {
			commandID = m.GetStringValue()
			continue
		}

		if m.GetName() == "Node Control/Lora Pairing" {

			duration = int(m.GetIntValue())

			continue
		}

	}

	if commandID == "" || duration == 0 {
		return
	}

	go n.loraThings.Paring(duration)

	response := &model.CommandResponse{
		CommandID: commandID,
		Code:      200,
		Timestamp: uint64(time.Now().UnixMicro()),
	}

	if payload, err := response.ToPayload(); err == nil {
		msg := &model.SpMessage{
			Topic:   fmt.Sprintf("spBv1.0/devices/NDATA/%v", n.id),
			Payload: payload,
		}

		n.dataChan <- msg

	}

}

func (n *Node) deleteDevice(p *model.Payload) {

	guid := ""
	for _, m := range p.Metrics {
		if m.GetName() == "device id" {
			guid = m.GetStringValue()
			break
		}
	}

	go n.canThings.Delete(guid)
	go n.serialThings.Delete(guid)
	go n.loraThings.Delete(guid)
}

func newRebirthMetric(ts uint64) *model.Payload_Metric {
	metric := &model.Payload_Metric{
		Name:      proto.String("Node Control/Rebirth"),
		Datatype:  proto.Uint32(uint32(model.DataType_Boolean)),
		Value:     &model.Payload_Metric_BooleanValue{BooleanValue: false},
		Timestamp: proto.Uint64(ts),
	}

	return metric
}

func newRebootMetric(ts uint64) *model.Payload_Metric {
	metric := &model.Payload_Metric{
		Name:      proto.String("Node Control/Reboot"),
		Datatype:  proto.Uint32(uint32(model.DataType_Boolean)),
		Value:     &model.Payload_Metric_BooleanValue{BooleanValue: false},
		Timestamp: proto.Uint64(ts),
	}

	return metric
}

func newPropertiesMetrics(ts uint64) []*model.Payload_Metric {

	var metrics []*model.Payload_Metric

	ip := &model.Payload_Metric{
		Name:      proto.String("Properties/IPv4"),
		Datatype:  proto.Uint32(uint32(model.DataType_String)),
		Value:     &model.Payload_Metric_StringValue{StringValue: system.IPv4()},
		Timestamp: proto.Uint64(ts),
	}

	version := &model.Payload_Metric{
		Name:      proto.String("Properties/Node Version"),
		Datatype:  proto.Uint32(uint32(model.DataType_String)),
		Value:     &model.Payload_Metric_StringValue{StringValue: system.NodeVersion()},
		Timestamp: proto.Uint64(ts),
	}

	metrics = append(metrics, ip, version)

	return metrics
}

func (n *Node) nbirth() *model.Payload {

	ts := uint64(time.Now().UnixMicro())

	bdSeq := n.sparkService.GetBdSeq()
	bdSeqMetric := model.NewBdSeqMetric(bdSeq, ts)
	rebirthMetric := newRebirthMetric(ts)

	properties := newPropertiesMetrics(ts)

	p := model.NewPayload()

	p.Metrics = append(p.Metrics, bdSeqMetric, rebirthMetric)
	p.Metrics = append(p.Metrics, properties...)

	return p

}

func (n Node) onConnectHandler(c mqtt.Client) {

	// primary host app STATE
	c.Subscribe("spBv1.0/STATE/+", 0, n.stateCallback)

	// NCMD
	c.Subscribe(fmt.Sprintf("spBv1.0/devices/NCMD/%v", n.id), 0, n.nodeCommandCallback)

	// DCMD
	c.Subscribe(fmt.Sprintf("spBv1.0/devices/DCMD/%v/+", n.id), 0, n.deviceCommandCallback)

	// lora
	c.Subscribe(fmt.Sprintf("spBv1.0/lora/DBIRTH/%v/+", n.id), 0, n.loraBirthCallback)

	// NBIRTH
	msg := &model.SpMessage{
		Topic:    fmt.Sprintf("spBv1.0/devices/NBIRTH/%v", n.id),
		Payload:  n.nbirth(),
		Retained: true,
	}

	n.dataChan <- msg

}

// NCMD
func (n *Node) nodeCommandCallback(c mqtt.Client, m mqtt.Message) {

	if !n.primaryHostAppOnline {
		return
	}

	p := &model.Payload{}
	err := proto.Unmarshal(m.Payload(), p)

	if err != nil {
		return
	}

	for _, m := range p.GetMetrics() {

		switch m.GetName() {
		case "Node Control/Rebirth":
			n.rebirth()
		case "Node Control/Reboot":
			n.reboot()
		case "Node Control/Create Device":
			n.createDevice(p)
		case "Node Control/Delete Device":
			n.deleteDevice(p)
		case "Node Control/Lora Pairing":
			n.loraPairing(p)
		}

	}
}

// DCMD
func (n *Node) deviceCommandCallback(c mqtt.Client, m mqtt.Message) {
	if !n.primaryHostAppOnline {
		return
	}

	guid := utils.GetTopicN(m.Topic(), 4)

	go n.canThings.Request(guid, m.Payload())
	go n.loraThings.Request(guid, m.Payload())
	go n.serialThings.Request(guid, m.Payload())

}

func (n *Node) loraBirthCallback(c mqtt.Client, m mqtt.Message) {
	if !n.primaryHostAppOnline {
		return
	}

	sn := utils.GetTopicN(m.Topic(), 4)
	guid := ""
	deviceType := 0
	converterType := 0

	p := &model.Payload{}
	err := proto.Unmarshal(m.Payload(), p)

	if err != nil {
		return
	}

	for _, metric := range p.GetMetrics() {

		switch metric.GetName() {
		case "guid":
			guid = metric.GetStringValue()
		case "device type":
			deviceType = int(metric.GetIntValue())
		case "converter type":
			converterType = int(metric.GetIntValue())
		}

	}

	n.loraThings.addLoraThing(guid, deviceType, sn, converterType)

}

func (n *Node) Run() {

	go n.canThings.Process()
	go n.loraThings.Process()
	go n.serialThings.Process()

	go n.sparkService.Run()

	defer n.db.Close()
	for msg := range n.dataChan {
		msg.Payload.Timestamp = proto.Uint64(uint64(time.Now().UnixMicro()))
		msg.Payload.Seq = proto.Uint64(uint64(n.seq))

		if payload, err := proto.Marshal(msg.Payload); err == nil {
			n.sparkService.Publish(msg.Topic, msg.Qos, msg.Retained, payload)
		}
	}
}

// todo
func (n *Node) rebirth() {

	// NBIRTH

	// DBIRTH
}

// todo
func (n *Node) reboot() {
}
