package model

type Heart struct {
	Conected bool `json:"connected"`

	last          uint16
	current       uint16
	statusChanged bool
}

func (h *Heart) HeartCheck() {

	var connected bool
	if h.current != h.last {
		connected = true
	} else {
		connected = false
	}

	h.statusChanged = (h.Conected != connected)

	h.last = h.current
	h.Conected = connected

}

func (h *Heart) HeartBeat() {

	h.current = h.current + 1
}

func (h *Heart) Changed() bool {
	return h.statusChanged
}
