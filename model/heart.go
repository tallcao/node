package model

type Heart struct {
	connected bool

	last          uint8
	current       uint8
	statusChanged bool
}

func (h *Heart) HeartCheck() {

	var connected bool
	if h.current != h.last {
		connected = true
	} else {
		connected = false
	}

	h.statusChanged = (h.connected != connected)

	h.last = h.current
	h.connected = connected

}

func (h *Heart) HeartBeat() {

	h.current = h.current + 1
}

func (h *Heart) ConnectedChanged() bool {
	return h.statusChanged
}

func (h *Heart) IsConnected() bool {
	return h.connected
}
