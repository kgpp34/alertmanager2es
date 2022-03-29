package model

import (
	log "github.com/sirupsen/logrus"
	"github.com/webdevops/alertmanager2es/utils"
)

type NamespaceLowHealthEvent struct {
}

func (event *NamespaceLowHealthEvent) HandleEvent(alert AlertmanagerEntry, url string) (bool, error) {
	log.Info("enter the namespace-low-health-event-handler")

	for _, alertEvent := range alert.Alerts {
		namespaceName := alertEvent.Labels["namespace"]

		if len(namespaceName) == 0 {
			log.Error("namespaceName is null string, please check")
			return false, nil
		}

		// construct post request
		h := utils.NewHttpSend(url)

		h.SetBody(map[string]string{
			"namespaceName": namespaceName,
		})

		// send request
		_, err := h.Post()
		if err != nil {
			log.Infof("post to k8s admin error : %v", err)
			return false, err
		}

	}
	return true, nil
}
