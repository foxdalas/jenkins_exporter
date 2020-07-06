package jobs_stats

import (
	"github.com/bndr/gojenkins"
	"github.com/prometheus/common/log"
	"sync"
)

type JobsStats struct {
	Name     string
	Branch   string
	Duration int64
}

type BranchStats struct {
	Name     string
	Duration int64
}

func Jobs(jenkins *gojenkins.Jenkins) ([]*gojenkins.Job, error) {
	return jenkins.GetAllJobs()
}

func GetBranches(jobs []*gojenkins.Job) []BranchStats {
	var wg sync.WaitGroup
	var stats []BranchStats
	var mux sync.RWMutex

	wg.Add(len(jobs))

	for i := 0; i < len(jobs); i++ {
		go func(job *gojenkins.Job) {
			defer wg.Done()
			success, err := job.GetLastSuccessfulBuild()
			if err != nil {
				log.Debugf("Job %s get latast successful build error: %s", job.GetName(), err)
				return
			}
			stat := BranchStats{
				Name:     success.Job.GetName(),
				Duration: success.GetDuration(),
			}
			mux.Lock()
			stats = append(stats, stat)
			mux.Unlock()
		}(jobs[i])
	}
	wg.Wait()
	return stats
}

func GetDuration(jobs []*gojenkins.Job) ([]JobsStats, error) {
	var stats []JobsStats
	var wg sync.WaitGroup
	var mux sync.RWMutex
	wg.Add(len(jobs))

	for i := 0; i < len(jobs); i++ {
		go func(job *gojenkins.Job) {
			defer wg.Done()
			//Get Job name
			log.Info(job.GetName())
			//Get branches
			branches, err := job.GetInnerJobs()
			if err != nil {
				log.Debugf("Job %s get inner jobs error: %s", job.GetName(), err)
				return
			}

			branchesStats := GetBranches(branches)
			for _, branchStat := range branchesStats {
				stat := JobsStats{
					Name:     job.GetName(),
					Branch:   branchStat.Name,
					Duration: branchStat.Duration,
				}
				mux.Lock()
				stats = append(stats, stat)
				mux.Unlock()
			}
		}(jobs[i])
	}
	wg.Wait()
	return stats, nil
}

func Get(jenkins *gojenkins.Jenkins) ([]JobsStats, error) {
	var stats []JobsStats

	jobs, err := Jobs(jenkins)
	if err != nil {
		return stats, err
	}
	return GetDuration(jobs)
}
