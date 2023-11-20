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
	"sync"
	"time"

	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/modules/keyvalue"
	"code.vikunja.io/web"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// SecondsUntilInactive defines the seconds until a user is considered inactive
const SecondsUntilInactive = 30

// ActiveUsersKey is the key used to store active users in redis
const ActiveUsersKey = `activeusers`

// ActiveUser defines an active user
type ActiveUser struct {
	UserID   int64
	LastSeen time.Time
}

type activeUsersMap map[int64]*ActiveUser

// ActiveUsers is the type used to save active users
type ActiveUsers struct {
	users activeUsersMap
	mutex *sync.Mutex
}

// activeUsers holds a map with all active users
var activeUsers *ActiveUsers

func init() {
	activeUsers = &ActiveUsers{
		users: make(map[int64]*ActiveUser),
		mutex: &sync.Mutex{},
	}
}

func setupActiveUsersMetric() {
	err := registry.Register(promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "vikunja_active_users",
		Help: "The number of users active within the last 30 seconds on this node",
	}, func() float64 {
		allActiveUsers, err := getActiveUsers()
		if err != nil {
			log.Error(err.Error())
		}
		if allActiveUsers == nil {
			return 0
		}
		activeUsersCount := 0
		for _, u := range allActiveUsers {
			if time.Since(u.LastSeen) < SecondsUntilInactive*time.Second {
				activeUsersCount++
			}
		}
		return float64(activeUsersCount)
	}))
	if err != nil {
		log.Criticalf("Could not register metrics for currently active users: %s", err)
	}
}

// SetUserActive sets a user as active and pushes it to redis
func SetUserActive(a web.Auth) (err error) {
	activeUsers.mutex.Lock()
	activeUsers.users[a.GetID()] = &ActiveUser{
		UserID:   a.GetID(),
		LastSeen: time.Now(),
	}
	activeUsers.mutex.Unlock()
	return PushActiveUsers()
}

// getActiveUsers returns the active users from redis
func getActiveUsers() (users activeUsersMap, err error) {
	users = activeUsersMap{}
	_, err = keyvalue.GetWithValue(ActiveUsersKey, &users)
	return
}

// PushActiveUsers pushed the content of the activeUsers map to redis
func PushActiveUsers() (err error) {
	activeUsers.mutex.Lock()
	defer activeUsers.mutex.Unlock()

	return keyvalue.Put(ActiveUsersKey, activeUsers.users)
}
