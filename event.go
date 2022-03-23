package main

type Event interface {
	HandleEvent(alert AlertmanagerEntry) (bool, error)
}
