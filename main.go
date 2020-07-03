package main

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"time"

	"github.com/foxdalas/jenkins_exporter/pkg/agents"
	"github.com/foxdalas/jenkins_exporter/pkg/jobs_stats"
	"github.com/foxdalas/jenkins_exporter/pkg/view"

	"github.com/bndr/gojenkins"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
)

const (
	namespace           = "jenkins"
	jenkinsPollInterval = 5 * time.Second
)

var (
	job_label = []string{"job"}
	computer_label = []string{"computer"}
	state_label = []string{"state"}
	view_label = []string{"view"}

	jobsStats []*jobs_stats.JobsStats
	jobsStatsLock bool
)



type Exporter struct {
	jenkins map[string]string

	up              *prometheus.Desc
	jobs            *prometheus.Desc
	queue           *prometheus.Desc
	total_executors *prometheus.Desc
	busy_executors  *prometheus.Desc
	computers       *prometheus.Desc
	computer_idle   *prometheus.Desc

	views *prometheus.Desc

	jnlp_agents *prometheus.Desc

	job_duration *prometheus.Desc

	computer_num_executors *prometheus.Desc
	computer_offline       *prometheus.Desc
}

func NewExporter(url string, username string, password string, timeout time.Duration) *Exporter {
	c := map[string]string{
		"url":      url,
		"username": username,
		"password": password,
	}

	return &Exporter{
		jenkins: c,

		up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "up"),
			"Could the jenkins server be reached.",
			nil,
			nil,
		),
		jobs: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "jobs_stats"),
			"Jobs count.",
			nil,
			nil,
		),
		job_duration: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "job_duration"),
			"Job duration.",
			job_label,
			nil,
		),
		queue: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "queue"),
			"Queue items count.",
			nil,
			nil,
		),
		total_executors: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "total_executors"),
			"Jenkins count Total Executors.",
			nil,
			nil,
		),
		busy_executors: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "busy_executors"),
			"Jenkins count Busy Executors.",
			nil,
			nil,
		),
		computers: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "computers"),
			"Build agents count.",
			nil,
			nil,
		),
		computer_idle: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "computer_idle"),
			"Jenkins computer state idle",
			computer_label,
			nil,
		),
		computer_offline: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "computer_offline"),
			"Jenkins computer state offline",
			computer_label,
			nil,
		),
		computer_num_executors: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "computer_num_executors"),
			"Jenkins computer count executors",
			computer_label,
			nil,
		),

		views: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "views"),
			"Jenkins Views Count",
			view_label,
			nil,
		),

		jnlp_agents: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "jnlp_agents"),
			"Jenkins JNLP Agents",
			state_label,
			nil,
		),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.up
	ch <- e.jobs
	ch <- e.queue
	ch <- e.computers
	ch <- e.computer_num_executors
	ch <- e.computer_offline
	ch <- e.views
	ch <- e.jnlp_agents
	ch <- e.job_duration
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	status := 1
	jenkins := gojenkins.CreateJenkins(nil, e.jenkins["url"], e.jenkins["username"], e.jenkins["password"])
	_, err := jenkins.Init()
	assertError(err, &status)

	jobs, err := jenkins.GetAllJobNames()
	assertError(err, &status)

	queue, err := jenkins.GetQueue()
	assertError(err, &status)

	views, err := view.Get(jenkins)
	assertError(err, &status)

	for _, view := range views {
		ch <- prometheus.MustNewConstMetric(e.views, prometheus.GaugeValue, float64(len(view.Jobs)), view.Name)
	}

	jnlp_agents, err := agents.Get(jenkins)
	assertError(err, &status)

	//Collect jobs
	go func()  {
		if !jobsStatsLock {
			jobsStatsLock = true
			log.Info("Collecting jobs ")
			queueJobsStats, err := jobs_stats.Get(jenkins)
			if err != nil {
				log.Error(err)
			}
			jobsStatsLock = false
			jobsStats = queueJobsStats
			log.Info("Finish collecting jobs")
		} else {
			log.Warn("Still collecting jobs")
		}
	}()

	log.Infof("Jobs %d", len(jobsStats))

	for _, jobStats := range jobsStats {
		ch <- prometheus.MustNewConstMetric(e.job_duration, prometheus.GaugeValue, float64(jobStats.Duration), jobStats.Name)
	}

	ch <- prometheus.MustNewConstMetric(e.jobs, prometheus.GaugeValue, float64(len(jobs)))
	ch <- prometheus.MustNewConstMetric(e.queue, prometheus.GaugeValue, float64(len(queue.Tasks())))
	ch <- prometheus.MustNewConstMetric(e.computers, prometheus.GaugeValue, jnlp_agents.Total)

	ch <- prometheus.MustNewConstMetric(e.jnlp_agents, prometheus.GaugeValue, jnlp_agents.Idle, "idle")
	ch <- prometheus.MustNewConstMetric(e.jnlp_agents, prometheus.GaugeValue, jnlp_agents.Online, "online")
	ch <- prometheus.MustNewConstMetric(e.jnlp_agents, prometheus.GaugeValue, jnlp_agents.Offline, "offline")
	ch <- prometheus.MustNewConstMetric(e.jnlp_agents, prometheus.GaugeValue, jnlp_agents.Busy, "busy")


	//
	//for _, computer := range computers {
	//	ch <- prometheus.MustNewConstMetric(e.computer_idle, prometheus.CounterValue, bool2float64(computer.Idle), computer.DisplayName)
	//	ch <- prometheus.MustNewConstMetric(e.computer_offline, prometheus.CounterValue, bool2float64(computer.Offline), computer.DisplayName)
	//	ch <- prometheus.MustNewConstMetric(e.computer_num_executors, prometheus.CounterValue, float64(computer.NumExecutors), computer.DisplayName)
	//}
	ch <- prometheus.MustNewConstMetric(e.up, prometheus.GaugeValue, float64(status))

}

func bool2float64(value bool) float64 {
	if value == true {
		return float64(1)
	} else {
		return float64(0)
	}
}

func assertError(err error, status *int) {
	if err != nil {
		*status = 0
		log.Errorf("Failed to collect jenkins data %s from jenkins", err)
	}
}

func main() {
	var (
		url           = kingpin.Flag("jenkins.url", "Jenkins server url.").Default("http://localhost:8080").String()
		username      = kingpin.Flag("jenkins.username", "Jenkins server username.").Default("admin").String()
		password      = kingpin.Flag("jenkins.password", "Jenkins server token/password.").Default("admin").String()
		timeout       = kingpin.Flag("jenkins.timeout", "Jenkins server connect timeout.").Default("30s").Duration()
		listenAddress = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9118").String()
		metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	)

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("jenkins_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting jenkins_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	prometheus.MustRegister(NewExporter(*url, *username, *password, *timeout))

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
      <head><title>Jenkins Exporter</title></head>
      <body>
      <h1>Jenkins Exporter</h1>
      <p><a href='` + *metricsPath + `'>Metrics</a></p>
      </body>
      </html>`))
	})

	log.Infoln("Starting HTTP server on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
