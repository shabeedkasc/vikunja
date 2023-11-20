// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public Licensee as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public Licensee for more details.
//
// You should have received a copy of the GNU Affero General Public Licensee
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package metrics

import (
	"strconv"

	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/modules/keyvalue"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// ProjectCountKey is the name of the key in which we save the project count
	ProjectCountKey = `projectcount`

	// UserCountKey is the name of the key we use to store total users in redis
	UserCountKey = `usercount`

	// TaskCountKey is the name of the key we use to store the amount of total tasks in redis
	TaskCountKey = `taskcount`

	// TeamCountKey is the name of the key we use to store the amount of total teams in redis
	TeamCountKey = `teamcount`
)

var registry *prometheus.Registry

func GetRegistry() *prometheus.Registry {
	if registry == nil {
		registry = prometheus.NewRegistry()
		registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
		registry.MustRegister(collectors.NewGoCollector())
	}

	return registry
}

// InitMetrics Initializes the metrics
func InitMetrics() {
	// init active users, sometimes we'll have garbage from previous runs in redis instead
	if err := PushActiveUsers(); err != nil {
		log.Fatalf("Could not set initial count for active users, error was %s", err)
	}

	GetRegistry()

	// Register total project count metric
	err := registry.Register(promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "vikunja_project_count",
		Help: "The number of projects on this instance",
	}, func() float64 {
		count, _ := GetCount(ProjectCountKey)
		return float64(count)
	}))
	if err != nil {
		log.Criticalf("Could not register metrics for %s: %s", ProjectCountKey, err)
	}

	// Register total user count metric
	err = registry.Register(promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "vikunja_user_count",
		Help: "The total number of users on this instance",
	}, func() float64 {
		count, _ := GetCount(UserCountKey)
		return float64(count)
	}))
	if err != nil {
		log.Criticalf("Could not register metrics for %s: %s", UserCountKey, err)
	}

	// Register total Tasks count metric
	err = registry.Register(promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "vikunja_task_count",
		Help: "The total number of tasks on this instance",
	}, func() float64 {
		count, _ := GetCount(TaskCountKey)
		return float64(count)
	}))
	if err != nil {
		log.Criticalf("Could not register metrics for %s: %s", TaskCountKey, err)
	}

	// Register total teams count metric
	err = registry.Register(promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "vikunja_team_count",
		Help: "The total number of teams on this instance",
	}, func() float64 {
		count, _ := GetCount(TeamCountKey)
		return float64(count)
	}))
	if err != nil {
		log.Criticalf("Could not register metrics for %s: %s", TeamCountKey, err)
	}

	setupActiveUsersMetric()
}

// GetCount returns the current count from keyvalue
func GetCount(key string) (count int64, err error) {
	cnt, exists, err := keyvalue.Get(key)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, nil
	}

	if s, is := cnt.(string); is {
		count, err = strconv.ParseInt(s, 10, 64)
	} else {
		count = cnt.(int64)
	}

	return
}

// SetCount sets the project count to a given value
func SetCount(count int64, key string) error {
	return keyvalue.Put(key, count)
}
