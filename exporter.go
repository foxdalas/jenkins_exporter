package main

import (
	"context"
	"errors"
	"github.com/bndr/gojenkins"
	"github.com/foxdalas/jenkins_exporter/pkg/agents"
	"github.com/foxdalas/jenkins_exporter/pkg/cache"
	"github.com/foxdalas/jenkins_exporter/pkg/jobs_stats"
	"github.com/foxdalas/jenkins_exporter/pkg/view"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

const (
	namespace                 = "jenkins"
	jenkinsPollInterval       = 5 * time.Second
	statusOk            int32 = 1
	statusFailed        int32 = 0
	statusFirstRun      int32 = -1
)

var (
	jobLabel      = []string{"job", "branch"}
	computerLabel = []string{"computer"}
	stateLabel    = []string{"state"}
	viewLabel     = []string{"view"}
)

type Exporter struct {
	jenkinsDetail map[string]string
	lastRunStatus int32
	runInterval   time.Duration
	context       context.Context
	cancelFunc    context.CancelFunc
	wg            sync.WaitGroup
	metricsCache  *cache.Cache
	metrics       exportedMetrics
}

type exportedMetrics struct {
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

func NewExporter(url string, username string, password string, interval, timeout time.Duration) *Exporter {
	c := map[string]string{
		"url":      url,
		"username": username,
		"password": password,
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Exporter{
		jenkinsDetail: c,
		runInterval:   interval,
		metricsCache:  cache.New(),
		cancelFunc:    cancel,
		lastRunStatus: statusFirstRun,
		context:       ctx,
		metrics: exportedMetrics{
			up: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "", "up"),
				"Could the jenkinsDetail server be reached.",
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
				jobLabel,
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
				computerLabel,
				nil,
			),
			computer_offline: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "", "computer_offline"),
				"Jenkins computer state offline",
				computerLabel,
				nil,
			),
			computer_num_executors: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "", "computer_num_executors"),
				"Jenkins computer count executors",
				computerLabel,
				nil,
			),

			views: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "", "views"),
				"Jenkins Views Count",
				viewLabel,
				nil,
			),

			jnlp_agents: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "", "jnlp_agents"),
				"Jenkins JNLP Agents",
				stateLabel,
				nil,
			),
		},
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.metrics.up
	ch <- e.metrics.jobs
	ch <- e.metrics.queue
	ch <- e.metrics.computers
	ch <- e.metrics.computer_num_executors
	ch <- e.metrics.computer_offline
	ch <- e.metrics.views
	ch <- e.metrics.jnlp_agents
	ch <- e.metrics.job_duration
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	lastRunStatus := atomic.LoadInt32(&e.lastRunStatus)
	ch <- prometheus.MustNewConstMetric(e.metrics.up, prometheus.GaugeValue, float64(lastRunStatus))

	cachedValues := e.metricsCache.GetValues()

	for _, view := range cachedValues.Views {
		ch <- prometheus.MustNewConstMetric(e.metrics.views, prometheus.GaugeValue, float64(len(view.Jobs)), view.Name)
	}

	for _, jobStat := range cachedValues.JobsStats {
		log.Infof("Job name: %s branch: %s with duration %d", jobStat.Name, jobStat.Branch, jobStat.Duration)
		ch <- prometheus.MustNewConstMetric(e.metrics.job_duration, prometheus.GaugeValue, float64(jobStat.Duration), jobStat.Name, jobStat.Branch)
	}

	ch <- prometheus.MustNewConstMetric(e.metrics.jobs, prometheus.GaugeValue, float64(cachedValues.JobsCount))
	ch <- prometheus.MustNewConstMetric(e.metrics.queue, prometheus.GaugeValue, float64(cachedValues.QueueTasks))
	ch <- prometheus.MustNewConstMetric(e.metrics.computers, prometheus.GaugeValue, cachedValues.Agents.Total)

	ch <- prometheus.MustNewConstMetric(e.metrics.jnlp_agents, prometheus.GaugeValue, cachedValues.Agents.Idle, "idle")
	ch <- prometheus.MustNewConstMetric(e.metrics.jnlp_agents, prometheus.GaugeValue, cachedValues.Agents.Online, "online")
	ch <- prometheus.MustNewConstMetric(e.metrics.jnlp_agents, prometheus.GaugeValue, cachedValues.Agents.Offline, "offline")
	ch <- prometheus.MustNewConstMetric(e.metrics.jnlp_agents, prometheus.GaugeValue, cachedValues.Agents.Busy, "busy")

	//
	//for _, computer := range computers {
	//	ch <- prometheus.MustNewConstMetric(e.computer_idle, prometheus.CounterValue, bool2float64(computer.Idle), computer.DisplayName)
	//	ch <- prometheus.MustNewConstMetric(e.computer_offline, prometheus.CounterValue, bool2float64(computer.Offline), computer.DisplayName)
	//	ch <- prometheus.MustNewConstMetric(e.computer_num_executors, prometheus.CounterValue, float64(computer.NumExecutors), computer.DisplayName)
	//}

}

func (e *Exporter) Stop() {
	e.cancelFunc()
	e.wg.Wait()
}

func (e *Exporter) Run() {

	go func() {
		e.wg.Add(1)
		defer e.wg.Done()
		e.runOnce()

		ticker := time.NewTicker(e.runInterval)
		for {
			select {
			case <-e.context.Done():
				return
			case <-ticker.C:
				e.runOnce()
			}
		}
	}()
}

func (e *Exporter) runOnce() {
	jenkins := gojenkins.CreateJenkins(nil, e.jenkinsDetail["url"], e.jenkinsDetail["username"], e.jenkinsDetail["password"])
	_, err := jenkins.Init()
	e.assertError(err)

	if !e.credentialsAreOk(jenkins) {
		return
	}

	log.Info("Retrieving job names")
	jobs, err := jenkins.GetAllJobNames()
	e.assertError(err)

	log.Info("Retrieving queues")
	queue, err := jenkins.GetQueue()
	e.assertError(err)

	log.Info("Retrieving views")
	views, err := view.Get(jenkins)
	e.assertError(err)

	log.Info("Retrieving jnlp agents")
	jnlpAgents, err := agents.Get(jenkins)
	e.assertError(err)

	log.Info("Retrieving job stats")
	jobsStats, err := jobs_stats.Get(jenkins)
	log.Infof("Got %d jobs", len(jobsStats))

	e.metricsCache.SetValues(cache.Values{
		JobsCount:  len(jobs),
		QueueTasks: len(queue.Tasks()),
		Views:      views,
		Agents:     jnlpAgents,
		JobsStats:  jobsStats,
	})

	atomic.StoreInt32(&e.lastRunStatus, statusOk)
}

func (e *Exporter) credentialsAreOk(jenkins *gojenkins.Jenkins) bool {
	info, err := jenkins.Info()
	e.assertError(err)
	if reflect.DeepEqual(info, &gojenkins.ExecutorResponse{}) {
		e.assertError(errors.New("your credentials are bullshit, fuck off and die"))
		return false
	}

	return true
}

func (e *Exporter) assertError(err error) {
	if err != nil {
		atomic.StoreInt32(&e.lastRunStatus, statusFailed)
		log.Errorf("Failed to collect jenkins data from jenkins, error: '%s'", err)
	}
}
