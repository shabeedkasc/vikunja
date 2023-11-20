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
	"reflect"
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"
	"github.com/stretchr/testify/assert"
)

func TestTeam_Create(t *testing.T) {
	doer := &user.User{
		ID:       1,
		Username: "user1",
	}
	t.Run("normal", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		team := &Team{
			Name:        "Testteam293",
			Description: "Lorem Ispum",
		}
		err := team.Create(s, doer)
		assert.NoError(t, err)
		err = s.Commit()
		assert.NoError(t, err)
		db.AssertExists(t, "teams", map[string]interface{}{
			"id":          team.ID,
			"name":        "Testteam293",
			"description": "Lorem Ispum",
		}, false)
	})
	t.Run("empty name", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		team := &Team{}
		err := team.Create(s, doer)
		assert.Error(t, err)
		assert.True(t, IsErrTeamNameCannotBeEmpty(err))
	})
}

func TestTeam_ReadOne(t *testing.T) {
	u := &user.User{ID: 1}

	t.Run("normal", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		team := &Team{ID: 1}
		err := team.ReadOne(s, u)
		assert.NoError(t, err)
		assert.Equal(t, "testteam1", team.Name)
		assert.Equal(t, "Lorem Ipsum", team.Description)
		assert.Equal(t, int64(1), team.CreatedBy.ID)
		assert.Equal(t, int64(1), team.CreatedByID)
	})
	t.Run("invalid id", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		team := &Team{ID: -1}
		err := team.ReadOne(s, u)
		assert.Error(t, err)
		assert.True(t, IsErrTeamDoesNotExist(err))
	})
	t.Run("nonexisting", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		team := &Team{ID: 99999}
		err := team.ReadOne(s, u)
		assert.Error(t, err)
		assert.True(t, IsErrTeamDoesNotExist(err))
	})
}

func TestTeam_ReadAll(t *testing.T) {
	doer := &user.User{ID: 1}
	t.Run("normal", func(t *testing.T) {
		s := db.NewSession()
		defer s.Close()

		team := &Team{}
		teams, _, _, err := team.ReadAll(s, doer, "", 1, 50)
		assert.NoError(t, err)
		assert.Equal(t, reflect.TypeOf(teams).Kind(), reflect.Slice)
		ts := reflect.ValueOf(teams)
		assert.Equal(t, 5, ts.Len())
	})
	t.Run("search", func(t *testing.T) {
		s := db.NewSession()
		defer s.Close()

		team := &Team{}
		teams, _, _, err := team.ReadAll(s, doer, "READ_only_on_project6", 1, 50)
		assert.NoError(t, err)
		assert.Equal(t, reflect.TypeOf(teams).Kind(), reflect.Slice)
		ts := teams.([]*Team)
		assert.Len(t, ts, 1)
		assert.Equal(t, int64(2), ts[0].ID)
	})
}

func TestTeam_Update(t *testing.T) {
	u := &user.User{ID: 1}

	t.Run("normal", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		team := &Team{
			ID:   1,
			Name: "SomethingNew",
		}
		err := team.Update(s, u)
		assert.NoError(t, err)
		err = s.Commit()
		assert.NoError(t, err)
		db.AssertExists(t, "teams", map[string]interface{}{
			"id":   team.ID,
			"name": "SomethingNew",
		}, false)
	})
	t.Run("empty name", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		team := &Team{
			ID:   1,
			Name: "",
		}
		err := team.Update(s, u)
		assert.Error(t, err)
		assert.True(t, IsErrTeamNameCannotBeEmpty(err))
	})
	t.Run("nonexisting", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		team := &Team{
			ID:   9999,
			Name: "SomethingNew",
		}
		err := team.Update(s, u)
		assert.Error(t, err)
		assert.True(t, IsErrTeamDoesNotExist(err))
	})
}

func TestTeam_Delete(t *testing.T) {
	u := &user.User{ID: 1}

	t.Run("normal", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		team := &Team{
			ID: 1,
		}
		err := team.Delete(s, u)
		assert.NoError(t, err)
		err = s.Commit()
		assert.NoError(t, err)
		db.AssertMissing(t, "teams", map[string]interface{}{
			"id": 1,
		})
	})
}

func TestIsErrInvalidRight(t *testing.T) {
	assert.NoError(t, RightAdmin.isValid())
	assert.NoError(t, RightRead.isValid())
	assert.NoError(t, RightWrite.isValid())

	// Check invalid
	var tr Right = 938
	err := tr.isValid()
	assert.Error(t, err)
	assert.True(t, IsErrInvalidRight(err))
}
