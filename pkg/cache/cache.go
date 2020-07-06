package cache

import (
	"github.com/foxdalas/jenkins_exporter/pkg/agents"
	"github.com/foxdalas/jenkins_exporter/pkg/jobs_stats"
	"github.com/foxdalas/jenkins_exporter/pkg/view"
	"sync"
)

type Cache struct {
	mu sync.RWMutex
	Values
}

func New() *Cache {
	return &Cache{}
}

type Values struct {
	JobsCount  int
	QueueTasks int
	Views      []view.View
	Agents     agents.Agents
	JobsStats  []jobs_stats.JobsStats
}

func (c *Cache) SetValues(v Values) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Values = v
}

func (c *Cache) GetValues() Values {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Values
}
