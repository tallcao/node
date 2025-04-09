module edge

go 1.21

toolchain go1.23.3

require github.com/eclipse/paho.mqtt.golang v1.5.0

require google.golang.org/protobuf v1.35.2 // indirect

require (
	github.com/godbus/dbus/v5 v5.1.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/mattn/go-sqlite3 v1.14.24
	golang.org/x/net v0.27.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
)
