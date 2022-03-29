package _interface

import "github.com/webdevops/alertmanager2es/model"

type Event interface {
	HandleEvent(alert model.AlertmanagerEntry, url string) (bool, error)
}
