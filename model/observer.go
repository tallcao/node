package model

type Observer interface {
	Update(*Payload)
	GetID() string
}
