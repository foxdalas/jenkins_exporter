package jobs_stats

import (
	"github.com/bndr/gojenkins"
	"github.com/prometheus/common/log"
	"sync"
)

type JobsStats struct {
	Name string
	Duration int64
}

func Jobs(jenkins *gojenkins.Jenkins) ([]*gojenkins.Job, error) {
	return jenkins.GetAllJobs()
}

func GetDuration(jobs []*gojenkins.Job) ([]*JobsStats, error) {
	var stats []*JobsStats
	var wg sync.WaitGroup
	var mux sync.Mutex
	wg.Add(len(jobs))


	for i := 0; i < len(jobs); i++ {
		go func(job *gojenkins.Job) {
			defer wg.Done()
			success, err  := job.GetLastSuccessfulBuild()
			//fmt.Println(job.GetName())
			if err != nil {
				log.Errorf("Problem with job %s error %s", job.GetName(), err)
				return
			}
			stat := &JobsStats{
				Name: job.GetName(),
				Duration: success.GetDuration(),
			}
			mux.Lock()
			stats = append(stats, stat)
			mux.Unlock()
		}(jobs[i])
	}
	return stats, nil
}

func Get(jenkins *gojenkins.Jenkins) ([]*JobsStats, error) {
	var stats []*JobsStats

	jobs, err := Jobs(jenkins)
	if err != nil {
		return stats, err
	}
	return GetDuration(jobs)
}
