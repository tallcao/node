package service

import (
	"crypto/tls"
	"crypto/x509"
	"edge/model"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MqttRemoteService struct {
	ClientId string
	CaFile   string
	Uri      string

	User     string
	Password string

	Command chan *model.MqttMsg

	// on connected
	OnConnectedCh chan struct{}

	mqttClient mqtt.Client
}

func (s MqttRemoteService) Publish(topic string, qos byte, retained bool, payload []byte) {
	if s.mqttClient != nil {
		s.mqttClient.Publish(topic, qos, retained, payload)
	}
}

func (s MqttRemoteService) commandCallback(c mqtt.Client, m mqtt.Message) {

	msg := &model.MqttMsg{
		Topic:   m.Topic(),
		Payload: m.Payload(),
	}
	go func() { s.Command <- msg }()
}

func (s MqttRemoteService) onConnectHandler(c mqtt.Client) {

	// topic := fmt.Sprintf("command//%v/req/#", s.ClientId)
	topic := "command//+/req/#"
	c.Subscribe(topic, 0, s.commandCallback)

	// if token := c.Subscribe(topic, 0, s.commandCallback); token.Wait() && token.Error() != nil {
	// 	return
	// }

	go func() { s.OnConnectedCh <- struct{}{} }()

}

func (s MqttRemoteService) newTLSConfig() *tls.Config {
	// Import trusted certificates from CAfile.pem.
	// Alternatively, manually add CA certificates to
	// default openssl CA bundle.
	certpool := x509.NewCertPool()
	pemCerts, err := os.ReadFile(s.CaFile)
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

func (s *MqttRemoteService) Run() {
	opts := mqtt.NewClientOptions().AddBroker(s.Uri).SetClientID(s.ClientId).SetOrderMatters(false).SetOnConnectHandler(s.onConnectHandler)
	// opts := mqtt.NewClientOptions().AddBroker(s.Uri).SetOrderMatters(false).SetOnConnectHandler(s.onConnectHandler)
	tlsConfig := s.newTLSConfig()
	opts.SetUsername(s.User).SetPassword(s.Password).SetTLSConfig(tlsConfig)

	c := mqtt.NewClient(opts)

	s.mqttClient = c

	token := c.Connect()

	for token.Wait() && token.Error() != nil {
		time.Sleep(10 * time.Second)
		token = c.Connect()
	}

	defer c.Disconnect(250)

	select {}

}
