package main

type startupManager interface {
	Enabled() (bool, error)
	SetEnabled(enabled bool) error
}
