package model

type Observer interface {
	Update(string, map[string]any)
	// GetID() string
}
