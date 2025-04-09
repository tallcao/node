package model

import (
	"encoding/json"
	"time"
)

type PassiveReportingDevice interface {
	StartLoopRequest()
	StopLoopRequest()
}

type PassiveReporting struct {
	Interval           uint `json:"interval"`
	isInLoopRequesting bool
}

type intervalData struct {
	Data uint `json:"data"`
}

type RequestCommand func(string, interface{})

func (i *PassiveReporting) StartLoopRequest(request RequestCommand, cmd string) {
	i.isInLoopRequesting = true
	go func() {

		for {
			if !i.isInLoopRequesting {
				break
			}
			t := time.Now()

			request(cmd, nil)

			d := time.Until(t.Add(time.Second * time.Duration(i.Interval)))

			time.Sleep(d)
		}
	}()
}

func (i *PassiveReporting) SetInterval(params interface{}) {
	if jsonData, err := json.Marshal(params); err == nil {
		p := &intervalData{}
		err := json.Unmarshal(jsonData, &p)
		if p != nil && err == nil {
			i.Interval = p.Data
		}
	}
}

func (i *PassiveReporting) StopLoopRequest() {
	i.isInLoopRequesting = false
}
