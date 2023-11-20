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

package models

import (
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"
	"github.com/stretchr/testify/assert"
)

func TestSubscriptionGetTypeFromString(t *testing.T) {
	t.Run("project", func(t *testing.T) {
		entityType := getEntityTypeFromString("project")
		assert.Equal(t, SubscriptionEntityType(SubscriptionEntityProject), entityType)
	})
	t.Run("task", func(t *testing.T) {
		entityType := getEntityTypeFromString("task")
		assert.Equal(t, SubscriptionEntityType(SubscriptionEntityTask), entityType)
	})
	t.Run("invalid", func(t *testing.T) {
		entityType := getEntityTypeFromString("someomejghsd")
		assert.Equal(t, SubscriptionEntityType(SubscriptionEntityUnknown), entityType)
	})
}

func TestSubscription_Create(t *testing.T) {
	u := &user.User{ID: 1}

	t.Run("normal", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		sb := &Subscription{
			Entity:   "task",
			EntityID: 1,
			UserID:   u.ID,
		}

		can, err := sb.CanCreate(s, u)
		assert.NoError(t, err)
		assert.True(t, can)

		err = sb.Create(s, u)
		assert.NoError(t, err)
		assert.NotNil(t, sb.User)

		db.AssertExists(t, "subscriptions", map[string]interface{}{
			"entity_type": 3,
			"entity_id":   1,
			"user_id":     u.ID,
		}, false)
	})
	t.Run("forbidden for link shares", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		linkShare := &LinkSharing{}

		sb := &Subscription{
			Entity:   "task",
			EntityID: 1,
			UserID:   u.ID,
		}

		can, err := sb.CanCreate(s, linkShare)
		assert.Error(t, err)
		assert.False(t, can)
	})
	t.Run("noneixsting project", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		sb := &Subscription{
			Entity:   "project",
			EntityID: 99999999,
			UserID:   u.ID,
		}

		can, err := sb.CanCreate(s, u)
		assert.Error(t, err)
		assert.True(t, IsErrProjectDoesNotExist(err))
		assert.False(t, can)
	})
	t.Run("noneixsting task", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		sb := &Subscription{
			Entity:   "task",
			EntityID: 99999999,
			UserID:   u.ID,
		}

		can, err := sb.CanCreate(s, u)
		assert.Error(t, err)
		assert.True(t, IsErrTaskDoesNotExist(err))
		assert.False(t, can)
	})
	t.Run("no rights to see project", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		sb := &Subscription{
			Entity:   "project",
			EntityID: 20,
			UserID:   u.ID,
		}

		can, err := sb.CanCreate(s, u)
		assert.NoError(t, err)
		assert.False(t, can)
	})
	t.Run("no rights to see task", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		sb := &Subscription{
			Entity:   "task",
			EntityID: 14,
			UserID:   u.ID,
		}

		can, err := sb.CanCreate(s, u)
		assert.NoError(t, err)
		assert.False(t, can)
	})
	t.Run("existing subscription for (entity_id, entity_type, user_id) ", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		sb := &Subscription{
			Entity:   "task",
			EntityID: 2,
			UserID:   u.ID,
		}

		can, err := sb.CanCreate(s, u)
		assert.NoError(t, err)
		assert.True(t, can)

		err = sb.Create(s, u)
		assert.Error(t, err)
		assert.True(t, IsErrSubscriptionAlreadyExists(err))
	})

	// TODO: Add tests to test triggering of notifications for subscribed things
}

func TestSubscription_Delete(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		u := &user.User{ID: 1}
		sb := &Subscription{
			Entity:   "task",
			EntityID: 2,
			UserID:   u.ID,
		}

		can, err := sb.CanDelete(s, u)
		assert.NoError(t, err)
		assert.True(t, can)

		err = sb.Delete(s, u)
		assert.NoError(t, err)
		db.AssertMissing(t, "subscriptions", map[string]interface{}{
			"entity_type": 3,
			"entity_id":   2,
			"user_id":     u.ID,
		})
	})
	t.Run("forbidden for link shares", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		linkShare := &LinkSharing{}

		sb := &Subscription{
			Entity:   "task",
			EntityID: 1,
			UserID:   1,
		}

		can, err := sb.CanDelete(s, linkShare)
		assert.Error(t, err)
		assert.False(t, can)
	})
	t.Run("not owner of the subscription", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		u := &user.User{ID: 2}
		sb := &Subscription{
			Entity:   "task",
			EntityID: 2,
			UserID:   u.ID,
		}

		can, err := sb.CanDelete(s, u)
		assert.NoError(t, err)
		assert.False(t, can)
	})
}

func TestSubscriptionGet(t *testing.T) {
	u := &user.User{ID: 6}

	t.Run("test each individually", func(t *testing.T) {
		t.Run("project", func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			sub, err := GetSubscription(s, SubscriptionEntityProject, 12, u)
			assert.NoError(t, err)
			assert.NotNil(t, sub)
			assert.Equal(t, int64(3), sub.ID)
		})
		t.Run("task", func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			sub, err := GetSubscription(s, SubscriptionEntityTask, 22, u)
			assert.NoError(t, err)
			assert.NotNil(t, sub)
			assert.Equal(t, int64(4), sub.ID)
		})
	})
	t.Run("inherited", func(t *testing.T) {
		t.Run("project from parent", func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			// Project 25 belongs to project 12 where user 6 has subscribed to
			sub, err := GetSubscription(s, SubscriptionEntityProject, 25, u)
			assert.NoError(t, err)
			assert.NotNil(t, sub)
			assert.Equal(t, int64(12), sub.EntityID)
			assert.Equal(t, int64(3), sub.ID)
		})
		t.Run("project from parent's parent", func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			// Project 26 belongs to project 25 which belongs to project 12 where user 6 has subscribed to
			sub, err := GetSubscription(s, SubscriptionEntityProject, 26, u)
			assert.NoError(t, err)
			assert.NotNil(t, sub)
			assert.Equal(t, int64(12), sub.EntityID)
			assert.Equal(t, int64(3), sub.ID)
		})
		t.Run("task from parent", func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			// Task 39 belongs to project 25 which belongs to project 12 where the user has subscribed
			sub, err := GetSubscription(s, SubscriptionEntityTask, 39, u)
			assert.NoError(t, err)
			assert.NotNil(t, sub)
			// assert.Equal(t, int64(2), sub.ID) TODO
		})
		t.Run("task from project", func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			// Task 21 belongs to project 32 which the user has subscribed to
			sub, err := GetSubscription(s, SubscriptionEntityTask, 21, u)
			assert.NoError(t, err)
			assert.NotNil(t, sub)
			assert.Equal(t, int64(8), sub.ID)
		})
	})
	t.Run("invalid type", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		_, err := GetSubscription(s, 2342, 21, u)
		assert.Error(t, err)
		assert.True(t, IsErrUnknownSubscriptionEntityType(err))
	})
}
