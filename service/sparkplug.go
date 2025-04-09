package service

import (
	"crypto/tls"
	"crypto/x509"
	"edge/model"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"google.golang.org/protobuf/proto"
)

type SparkplugService struct {

	// mqtt option
	id     string
	cafile string
	uri    string

	mqttClient mqtt.Client

	onConn mqtt.OnConnectHandler

	bdSeq byte
}

func NewSparkplugService(id string, ca string, uri string) *SparkplugService {
	return &SparkplugService{
		id:     id,
		cafile: ca,
		uri:    uri,
	}
}
func (n SparkplugService) newTLSConfig() *tls.Config {
	// Import trusted certificates from CAfile.pem.
	// Alternatively, manually add CA certificates to
	// default openssl CA bundle.
	certpool := x509.NewCertPool()
	pemCerts, err := os.ReadFile(n.cafile)
	if err == nil {
		certpool.AppendCertsFromPEM(pemCerts)
	}

	// Import client certificate/key pair
	// cert, err := tls.LoadX509KeyPair("samplecerts/client-crt.pem", "samplecerts/client-key.pem")
	// if err != nil {
	// 	panic(err)
	// }

	// Create tls.Config with desired tls properties
	return &tls.Config{
		// RootCAs = certs used to verify server cert.
		RootCAs: certpool,
		// ClientAuth = whether to request cert from server.
		// Since the server is set up for SSL, this happens
		// anyways.
		ClientAuth: tls.NoClientCert,
		// ClientCAs = certs used to validate client cert.
		ClientCAs: nil,
		// InsecureSkipVerify = verify that cert contents
		// match server. IP matches what is in cert etc.
		InsecureSkipVerify: true,
		// Certificates = list of certs client sends to server.
		// Certificates: []tls.Certificate{cert},
	}
}

func (s *SparkplugService) GetBdSeq() byte {

	return s.bdSeq
}
func (s *SparkplugService) Publish(topic string, qos byte, retained bool, payload any) {

	s.mqttClient.Publish(topic, qos, retained, payload)
}

func (s *SparkplugService) SetOnConn(handler mqtt.OnConnectHandler) {
	s.onConn = handler
}
func (s *SparkplugService) onReconnectHandler(c mqtt.Client, opt *mqtt.ClientOptions) {

	ts := uint64(time.Now().UnixMicro())

	bdSeqMetric := model.NewBdSeqMetric(s.bdSeq, ts)

	p := &model.Payload{
		Timestamp: proto.Uint64(ts),
	}
	p.Metrics = append(p.Metrics, bdSeqMetric)

	topic := fmt.Sprintf("spBv1.0/devices/NDEATH/%v", s.id)

	if payload, err := proto.Marshal(p); err == nil {
		opt.SetBinaryWill(topic, payload, 1, false)

	}

	// todo
	// n.bdSeq += 1

}

func (s *SparkplugService) Run() {

	opts := mqtt.NewClientOptions().SetClientID(s.id).AddBroker(s.uri).SetOrderMatters(false)
	opts.OnConnect = s.onConn

	ts := uint64(time.Now().UnixMicro())

	bdSeqMetric := model.NewBdSeqMetric(s.bdSeq, ts)

	p := &model.Payload{
		Timestamp: proto.Uint64(ts),
	}
	p.Metrics = append(p.Metrics, bdSeqMetric)

	topic := fmt.Sprintf("spBv1.0/devices/NDEATH/%v", s.id)

	if payload, err := proto.Marshal(p); err == nil {
		opts.SetBinaryWill(topic, payload, 1, false)

	}

	tlsConfig := s.newTLSConfig()
	opts.SetTLSConfig(tlsConfig)

	c := mqtt.NewClient(opts)
	token := c.Connect()

	s.mqttClient = c

	for token.Wait() && token.Error() != nil {
		time.Sleep(10 * time.Second)
		token = c.Connect()
	}

	defer c.Disconnect(250)
	select {}

}
