package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/webdevops/alertmanager2es/utils"
)

const k8sAdminUrl = "http://172.31.199.48:8666/k8s-admin-proc/namespace/dealPodRestartTooManyHandler"

type PodRestartToManyEvent struct {
}

type PodRestartTooManyDTO struct {
	Namespace string `json:"namespace"`
	PodName   string `json:"podName"`
}

func (event *PodRestartToManyEvent) HandleEvent(alert AlertmanagerEntry) (bool, error) {
	// get alert message
	for _, alertEvent := range alert.Alerts {
		namespaceName := alertEvent.Labels["namespace"]
		podName := alertEvent.Labels["pod"]
		if len(podName) == 0 || len(namespaceName) == 0 {
			log.Error("podName or namespaceName is null, return")
			return false, nil
		}

		// construct post request
		h := utils.NewHttpSend(k8sAdminUrl)

		podRestartTooManyDTO := new(PodRestartTooManyDTO)
		podRestartTooManyDTO.Namespace = namespaceName
		podRestartTooManyDTO.PodName = podName

		//serialize
		// jsonData, err := json.Marshal(podRestartTooManyDTO)
		// if err != nil {
		// 	log.Error("serialize podRestartTooManyDTO failed")
		// }

		h.SetBody(map[string]string{
			"podName":       podName,
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
