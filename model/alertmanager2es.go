package model

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/webdevops/alertmanager2es/utils"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const (
	supportedWebhookVersion = "4"
	alertLabel              = "alertname"
)

var (
	k8sAdminPodRestartUrl         = ""
	k8sAdminNamespaceLowHealthUrl = ""
	//k8sTransFactory               sync.Once
	//eventFatory                   = make(map[string]_interface.Event)
)

type (
	AlertmanagerElasticsearchExporter struct {
		elasticSearchClient    *elasticsearch.Client
		elasticsearchIndexName string

		prometheus struct {
			alertsReceived   *prometheus.CounterVec
			alertsInvalid    *prometheus.CounterVec
			alertsSuccessful *prometheus.CounterVec
		}
	}

	AlertmanagerEntry struct {
		Alerts []struct {
			Annotations  map[string]string `json:"annotations"`
			EndsAt       time.Time         `json:"endsAt"`
			GeneratorURL string            `json:"generatorURL"`
			Labels       map[string]string `json:"labels"`
			StartsAt     time.Time         `json:"startsAt"`
			Status       string            `json:"status"`
		} `json:"alerts"`
		CommonAnnotations map[string]string `json:"commonAnnotations"`
		CommonLabels      map[string]string `json:"commonLabels"`
		ExternalURL       string            `json:"externalURL"`
		GroupLabels       map[string]string `json:"groupLabels"`
		Receiver          string            `json:"receiver"`
		Status            string            `json:"status"`
		Version           string            `json:"version"`
		GroupKey          string            `json:"groupKey"`

		// Timestamp records when the alert notification was received
		Timestamp string `json:"@timestamp"`
	}
)

func (e *AlertmanagerElasticsearchExporter) Init() {
	e.prometheus.alertsReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alertmanager2es_alerts_received",
			Help: "alertmanager2es received alerts",
		},
		[]string{},
	)
	prometheus.MustRegister(e.prometheus.alertsReceived)

	e.prometheus.alertsInvalid = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alertmanager2es_alerts_invalid",
			Help: "alertmanager2es invalid alerts",
		},
		[]string{},
	)
	prometheus.MustRegister(e.prometheus.alertsInvalid)

	e.prometheus.alertsSuccessful = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alertmanager2es_alerts_successful",
			Help: "alertmanager2es successful stored alerts",
		},
		[]string{},
	)
	prometheus.MustRegister(e.prometheus.alertsSuccessful)

	e.InitParser()
}

func (e *AlertmanagerElasticsearchExporter) ConnectElasticsearch(cfg elasticsearch.Config, indexName string) {
	var err error
	e.elasticSearchClient, err = elasticsearch.NewClient(cfg)
	if err != nil {
		panic(err)
	}

	tries := 0
	for {
		_, err = e.elasticSearchClient.Info()
		if err != nil {
			tries++
			if tries >= 5 {
				panic(err)
			} else {
				log.Info("Failed to connect to ES, retry...")
				time.Sleep(5 * time.Second)
				continue
			}
		}

		break
	}

	e.elasticsearchIndexName = indexName
}

func (e *AlertmanagerElasticsearchExporter) buildIndexName(createTime time.Time) string {
	ret := e.elasticsearchIndexName

	ret = strings.Replace(ret, "%y", createTime.Format("2006"), -1)
	ret = strings.Replace(ret, "%m", createTime.Format("01"), -1)
	ret = strings.Replace(ret, "%d", createTime.Format("02"), -1)

	return ret
}

func (e *AlertmanagerElasticsearchExporter) HttpHandler(w http.ResponseWriter, r *http.Request) {
	e.prometheus.alertsReceived.WithLabelValues().Inc()

	if r.Body == nil {
		e.prometheus.alertsInvalid.WithLabelValues().Inc()
		err := errors.New("got empty request body")
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Error(err)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		e.prometheus.alertsInvalid.WithLabelValues().Inc()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}
	defer r.Body.Close()

	var msg AlertmanagerEntry
	err = json.Unmarshal(b, &msg)
	if err != nil {
		e.prometheus.alertsInvalid.WithLabelValues().Inc()
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Error(err)
		return
	}

	if msg.Version != supportedWebhookVersion {
		e.prometheus.alertsInvalid.WithLabelValues().Inc()
		err := fmt.Errorf("do not understand webhook version %q, only version %q is supported", msg.Version, supportedWebhookVersion)
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Error(err)
		return
	}

	now := time.Now()
	msg.Timestamp = now.Format(time.RFC3339)

	incidentJson, _ := json.Marshal(msg)

	req := esapi.IndexRequest{
		Index: e.buildIndexName(now),
		Body:  bytes.NewReader(incidentJson),
	}
	res, err := req.Do(context.Background(), e.elasticSearchClient)
	if err != nil {
		e.prometheus.alertsInvalid.WithLabelValues().Inc()
		err := fmt.Errorf("unable to insert document in elasticsearch")
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Error(err)
		return
	}

	defer res.Body.Close()

	log.Debugf("received and stored alert: %v", msg.CommonLabels)
	e.prometheus.alertsSuccessful.WithLabelValues().Inc()

	e.k8sAdminTransHandler(msg)
}

func (e *AlertmanagerElasticsearchExporter) k8sAdminTransHandler(msg AlertmanagerEntry) {

	log.Info("get k8s admin url success")

	// send request to k8s-admin, 这一块后面会抽成工厂模式
	if msg.CommonLabels[alertLabel] == "PodRestartTooMany>20" {
		podRestartToManyEvent := new(PodRestartToManyEvent)

		_, err := podRestartToManyEvent.HandleEvent(msg, k8sAdminPodRestartUrl)
		if err != nil {
			log.Error(err)
		}
	}

	if msg.CommonLabels[alertLabel] == "NamespaceLowHealthLevel" {
		namespaceLowHealthEvent := new(NamespaceLowHealthEvent)

		_, err := namespaceLowHealthEvent.HandleEvent(msg, k8sAdminNamespaceLowHealthUrl)
		if err != nil {
			log.Error(err)
		}
	}
}

func (e *AlertmanagerElasticsearchExporter) InitParser() {
	log.Info("start init Ini Parser")
	path, err := exec.LookPath(os.Args[0])
	if err != nil {
		log.Error(err)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		log.Error(err)
	}

	println(abs)

	parser := utils.IniParser{}
	if err := parser.Load("request.ini"); err != nil {
		log.Error(err)
	}

	k8sAdminPodRestartUrl = parser.GetString("k8s-admin", "PodRestartUrl")
	//log.Infof("get the k8s-admin-pod-restart-handler-url : %s", k8sAdminPodRestartUrl)

	k8sAdminNamespaceLowHealthUrl = parser.GetString("k8s-admin", "NamespaceLowHealthUrl")
	//log.Infof("get the k8s-admin-namespace-low-health-handler-url : %s", k8sAdminNamespaceLowHealthUrl)
}
