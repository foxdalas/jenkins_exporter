package view

import "github.com/bndr/gojenkins"

type View struct {
	Name string
	Jobs []string
}

func Get(jenkins *gojenkins.Jenkins) ([]*View, error) {
	var viewsData []*View

	views, err := jenkins.GetAllViews()
	if err != nil {
		return viewsData, err
	}

	for _, view := range views {
		var jobs []string
		for _, job := range view.GetJobs() {
			jobs = append(jobs, job.Name)
		}

		viewData := &View{
			Name: view.GetName(),
			Jobs: jobs,
		}

		viewsData = append(viewsData,viewData)
	}

	return viewsData, err
}
