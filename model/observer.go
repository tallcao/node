package model

type Observer interface {
	Update(string, *Payload)
	// GetID() string
}
