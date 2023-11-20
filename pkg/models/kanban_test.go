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

	"xorm.io/xorm"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"
	"github.com/stretchr/testify/assert"
)

func TestBucket_ReadAll(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		testuser := &user.User{ID: 1}
		b := &Bucket{ProjectID: 1}
		bucketsInterface, _, _, err := b.ReadAll(s, testuser, "", 0, 0)
		assert.NoError(t, err)

		buckets, is := bucketsInterface.([]*Bucket)
		assert.True(t, is)

		// Assert that we have a user for each bucket
		assert.Equal(t, testuser.ID, buckets[0].CreatedBy.ID)
		assert.Equal(t, testuser.ID, buckets[1].CreatedBy.ID)
		assert.Equal(t, testuser.ID, buckets[2].CreatedBy.ID)

		// Assert our three test buckets
		assert.Len(t, buckets, 3)

		// Assert all tasks are in the right bucket
		assert.Len(t, buckets[0].Tasks, 12)
		assert.Len(t, buckets[1].Tasks, 3)
		assert.Len(t, buckets[2].Tasks, 3)

		// Assert we have bucket 1, 2, 3 but not 4 (that belongs to a different project) and their position
		assert.Equal(t, int64(1), buckets[0].ID)
		assert.Equal(t, int64(2), buckets[1].ID)
		assert.Equal(t, int64(3), buckets[2].ID)

		// Kinda assert all tasks are in the right buckets
		assert.Equal(t, int64(1), buckets[0].Tasks[0].BucketID)
		assert.Equal(t, int64(1), buckets[0].Tasks[1].BucketID)

		assert.Equal(t, int64(2), buckets[1].Tasks[0].BucketID)
		assert.Equal(t, int64(2), buckets[1].Tasks[1].BucketID)
		assert.Equal(t, int64(2), buckets[1].Tasks[2].BucketID)

		assert.Equal(t, int64(3), buckets[2].Tasks[0].BucketID)
		assert.Equal(t, int64(3), buckets[2].Tasks[1].BucketID)
		assert.Equal(t, int64(3), buckets[2].Tasks[2].BucketID)
	})
	t.Run("filtered", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		testuser := &user.User{ID: 1}
		b := &Bucket{
			ProjectID: 1,
			TaskCollection: TaskCollection{
				FilterBy:         []string{"title"},
				FilterComparator: []string{"like"},
				FilterValue:      []string{"done"},
			},
		}
		bucketsInterface, _, _, err := b.ReadAll(s, testuser, "", -1, 0)
		assert.NoError(t, err)

		buckets := bucketsInterface.([]*Bucket)
		assert.Len(t, buckets, 3)
		assert.Equal(t, int64(2), buckets[0].Tasks[0].ID)
		assert.Equal(t, int64(33), buckets[0].Tasks[1].ID)
	})
	t.Run("accessed by link share", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		linkShare := &LinkSharing{
			ID:        1,
			ProjectID: 1,
			Right:     RightRead,
		}
		b := &Bucket{ProjectID: 1}
		result, _, _, err := b.ReadAll(s, linkShare, "", 0, 0)
		assert.NoError(t, err)
		buckets, _ := result.([]*Bucket)
		assert.Len(t, buckets, 3)
		assert.NotNil(t, buckets[0].CreatedBy)
		assert.Equal(t, int64(1), buckets[0].CreatedByID)
	})
	t.Run("created by link share", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		testuser := &user.User{ID: 12}
		b := &Bucket{ProjectID: 23}
		result, _, _, err := b.ReadAll(s, testuser, "", 0, 0)
		assert.NoError(t, err)
		buckets, _ := result.([]*Bucket)
		assert.Len(t, buckets, 1)
		assert.NotNil(t, buckets[0].CreatedBy)
		assert.Equal(t, int64(-2), buckets[0].CreatedByID)
	})
}

func TestBucket_Delete(t *testing.T) {
	user := &user.User{ID: 1}

	t.Run("normal", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		b := &Bucket{
			ID:        2, // The second bucket only has 3 tasks
			ProjectID: 1,
		}
		err := b.Delete(s, user)
		assert.NoError(t, err)
		err = s.Commit()
		assert.NoError(t, err)

		// Assert all tasks have been moved to bucket 1 as that one is the first
		tasks := []*Task{}
		err = s.Where("bucket_id = ?", 1).Find(&tasks)
		assert.NoError(t, err)
		assert.Len(t, tasks, 15)
		db.AssertMissing(t, "buckets", map[string]interface{}{
			"id":         2,
			"project_id": 1,
		})
	})
	t.Run("last bucket in project", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		b := &Bucket{
			ID:        34,
			ProjectID: 18,
		}
		err := b.Delete(s, user)
		assert.Error(t, err)
		assert.True(t, IsErrCannotRemoveLastBucket(err))
		err = s.Commit()
		assert.NoError(t, err)

		db.AssertExists(t, "buckets", map[string]interface{}{
			"id":         34,
			"project_id": 18,
		}, false)
	})
}

func TestBucket_Update(t *testing.T) {

	testAndAssertBucketUpdate := func(t *testing.T, b *Bucket, s *xorm.Session) {
		err := b.Update(s, &user.User{ID: 1})
		assert.NoError(t, err)

		err = s.Commit()
		assert.NoError(t, err)

		db.AssertExists(t, "buckets", map[string]interface{}{
			"id":    1,
			"title": b.Title,
			"limit": b.Limit,
		}, false)
	}

	t.Run("normal", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		b := &Bucket{
			ID:    1,
			Title: "New Name",
			Limit: 2,
		}

		testAndAssertBucketUpdate(t, b, s)
	})
	t.Run("reset limit", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		b := &Bucket{
			ID:    1,
			Title: "testbucket1",
			Limit: 0,
		}

		testAndAssertBucketUpdate(t, b, s)
	})
}
