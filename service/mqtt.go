package service

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	// DefaultMqttService is the shared instance
	DefaultMqttService *MqttService
	once               sync.Once
)

type MqttService struct {
	id string

	client mqtt.Client

	mu       sync.Mutex                     // 保护订阅主题列表
	topics   map[string]byte                // 存储要订阅的主题及其 QoS
	handlers map[string]mqtt.MessageHandler // 存储特定主题的处理器
	running  bool                           // 服务是否正在运行

	onConnectHandlers []mqtt.OnConnectHandler
}

// InitSparkplugService initializes the default SparkplugService instance
func InitMqttService(id, brokerURL, caFile string) error {
	once.Do(func() {
		DefaultMqttService = NewMqttService(id, brokerURL, caFile)
	})
	return nil
}

// GetSparkplugService returns the default SparkplugService instance
func GetMqttService() *MqttService {
	return DefaultMqttService
}

func tlsConfig(caFile string) *tls.Config {
	// Skip TLS if no CA file provided
	if caFile == "" {
		return nil
	}

	// Load CA cert
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		log.Printf("Error loading CA file: %v", err)
		return nil
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		log.Printf("Error parsing CA certificate")
		return nil
	}

	return &tls.Config{
		RootCAs: caCertPool,

		ClientAuth: tls.NoClientCert,
		ClientCAs:  nil,

		// MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
	}
}

// NewMqttClientService 创建一个新的 MQTT 客户端服务实例
func NewMqttService(id, brokerURL, caFile string) *MqttService {

	opts := mqtt.NewClientOptions().AddBroker(brokerURL).SetClientID(id).SetOrderMatters(false)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetConnectRetry(true) // 启用自动重连

	opts.SetTLSConfig(tlsConfig(caFile))

	service := &MqttService{
		id:                id,
		topics:            make(map[string]byte),
		handlers:          make(map[string]mqtt.MessageHandler),
		onConnectHandlers: make([]mqtt.OnConnectHandler, 0),
	}

	// 设置 OnConnect 回调
	opts.SetOnConnectHandler(service.onConnectHandler)

	// node connect death
	opts.SetWill(fmt.Sprintf("node/%v/connected", id), "false", 1, true)

	service.client = mqtt.NewClient(opts)

	return service
}

func (s *MqttService) DeleteSubscriptionTopic(topic string) {

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.topics, topic)
	if s.running && s.client.IsConnected() {
		// 如果服务正在运行且已连接，立即取消订阅
		token := s.client.Unsubscribe(topic)
		token.Wait()
		if token.Error() != nil {
			log.Printf("ERROR: Failed to unsubscribe from topic '%s': %v\n", topic, token.Error())
		} else {
			log.Printf("Unsubscribed from topic: '%s'\n", topic)
		}
	}

	delete(s.handlers, topic)

}

// 这个方法可以在服务启动前或运行中调用
func (s *MqttService) AddSubscriptionTopic(topic string, qos byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.topics[topic] = qos

	if s.running && s.client.IsConnected() {

		// 如果服务正在运行且已连接，立即订阅
		s.subscribeToTopic(topic, qos)
	}
}

// 添加注册处理器的方法
func (s *MqttService) AddTopicHandler(topic string, handler mqtt.MessageHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[topic] = handler
}
func (s *MqttService) AddConnectHandler(handler mqtt.OnConnectHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onConnectHandlers = append(s.onConnectHandlers, handler)
}

// subscribeToTopic 内部函数，执行单个主题的订阅
func (s *MqttService) subscribeToTopic(topic string, qos byte) {

	handler, exists := s.handlers[topic]
	if !exists {
		log.Printf("ERROR: No handler registered for topic '%s'\n", topic)
		return
	}

	token := s.client.Subscribe(topic, qos, handler)
	token.Wait()
	if token.Error() != nil {
		log.Printf("ERROR: Failed to subscribe to topic '%s': %v\n", topic, token.Error())
	} else {
		log.Printf("Subscribed to topic: '%s' (QoS %d)\n", topic, qos)
	}

}

// onConnectHandler 是 MQTT 客户端连接成功时的回调
func (s *MqttService) onConnectHandler(client mqtt.Client) {
	log.Println("MQTT Client Connected!")

	s.mu.Lock()
	handlers := make([]mqtt.OnConnectHandler, len(s.onConnectHandlers))
	copy(handlers, s.onConnectHandlers)
	s.mu.Unlock()

	for _, handler := range handlers {
		if handler != nil {
			handler(client)
		}
	}
	// Handle topic subscriptions
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.topics) > 0 {
		log.Printf("Resubscribing to %d topics...\n", len(s.topics))
		for topic, qos := range s.topics {
			s.subscribeToTopic(topic, qos)
		}
	} else {
		log.Println("No topics registered for subscription.")
	}

}

func (s *MqttService) Start() error {

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return fmt.Errorf("MQTT Client Service is already running")
	}
	log.Println("Starting MQTT Client Service...")

	if token := s.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect MQTT client: %w", token.Error())
	}
	s.running = true
	log.Println("MQTT Client Service started.")
	return nil
}

func (s *MqttService) Stop() {
	s.client.Publish(fmt.Sprintf("node/%v/connected", s.id), 1, true, "false")

	log.Println("Stopping MQTT Client Service...")
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
	if s.client.IsConnected() {
		s.client.Disconnect(250)
	}
	log.Println("MQTT Client Service stopped.")
}

func (s *MqttService) PublishMessage(topic string, qos byte, retained bool, payload any) error {
	if !s.client.IsConnected() {
		return fmt.Errorf("MQTT client not connected, cannot publish")
	}
	token := s.client.Publish(topic, qos, retained, payload)
	token.Wait()
	if token.Error() != nil {
		return fmt.Errorf("failed to publish message to topic '%s': %w", topic, token.Error())
	}
	// log.Printf("Published message to topic: %s\n", topic)
	return nil
}

func (s *MqttService) GetClient() mqtt.Client {
	return s.client
}
