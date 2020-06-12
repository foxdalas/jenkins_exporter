package agents

import "github.com/bndr/gojenkins"

type Agents struct {
	Total float64

	Idle float64
	Busy float64

	Online float64
	Offline float64
}

func Get(jenkins *gojenkins.Jenkins) (*Agents, error){
	agents := &Agents{
		Total: 0.0,
		Idle: 0.0,
		Busy: 0.0,
		Online: 0.0,
		Offline: 0.0,
	}

	nodes, err := jenkins.GetAllNodes()
	if err != nil {
		return agents, err
	}

	for _, node := range nodes {
		online, err := node.IsOnline()
		if err != nil {
			return agents, err
		}
		if online {
			agents.Online += 1
		} else {
			agents.Offline += 1
		}
		node.IsIdle()
		jnlp, err := node.IsJnlpAgent()
		if err != nil {
			return agents, err
		}
		if jnlp {
			agents.Total += 1
		}

		idle, err := node.IsIdle()
		if err != nil {
			return agents, err
		}
		if idle && online {
			agents.Idle += 1
		}
	}
	agents.Busy = agents.Total - agents.Idle

	return agents, err
}
