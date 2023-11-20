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
	"time"

	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/web"
	"xorm.io/xorm"
)

// SavedFilter represents a saved bunch of filters
type SavedFilter struct {
	// The unique numeric id of this saved filter
	ID int64 `xorm:"autoincr not null unique pk" json:"id" param:"filter"`
	// The actual filters this filter contains
	Filters *TaskCollection `xorm:"JSON not null" json:"filters" valid:"required"`
	// The title of the filter.
	Title string `xorm:"varchar(250) not null" json:"title" valid:"required,runelength(1|250)" minLength:"1" maxLength:"250"`
	// The description of the filter
	Description string `xorm:"longtext null" json:"description"`
	OwnerID     int64  `xorm:"bigint not null INDEX" json:"-"`

	// The user who owns this filter
	Owner *user.User `xorm:"-" json:"owner" valid:"-"`

	// True if the filter is a favorite. Favorite filters show up in a separate parent project together with favorite projects.
	IsFavorite bool `xorm:"default false" json:"is_favorite"`

	// A timestamp when this filter was created. You cannot change this value.
	Created time.Time `xorm:"created not null" json:"created"`
	// A timestamp when this filter was last updated. You cannot change this value.
	Updated time.Time `xorm:"updated not null" json:"updated"`

	web.CRUDable `xorm:"-" json:"-"`
	web.Rights   `xorm:"-" json:"-"`
}

// TableName returns a better table name for saved filters
func (sf *SavedFilter) TableName() string {
	return "saved_filters"
}

func (sf *SavedFilter) getTaskCollection() *TaskCollection {
	// We're resetting the projectID to return tasks from all projects
	sf.Filters.ProjectID = 0
	return sf.Filters
}

// Returns the saved filter ID from a project ID. Will not check if the filter actually exists.
// If the returned ID is zero, means that it is probably invalid.
func getSavedFilterIDFromProjectID(projectID int64) (filterID int64) {
	// We get the id of the saved filter by multiplying the ProjectID with -1 and subtracting one
	filterID = projectID*-1 - 1
	// FilterIDs from projectIDs are always positive
	if filterID < 0 {
		filterID = 0
	}
	return
}

func getProjectIDFromSavedFilterID(filterID int64) (projectID int64) {
	projectID = filterID*-1 - 1
	// ProjectIDs from saved filters are always negative
	if projectID > 0 {
		projectID = 0
	}
	return
}

func getSavedFiltersForUser(s *xorm.Session, auth web.Auth) (filters []*SavedFilter, err error) {
	// Link shares can't view or modify saved filters, therefore we can error out right away
	if _, is := auth.(*LinkSharing); is {
		return nil, ErrSavedFilterNotAvailableForLinkShare{LinkShareID: auth.GetID()}
	}

	err = s.Where("owner_id = ?", auth.GetID()).Find(&filters)
	return
}

func (sf *SavedFilter) toProject() *Project {
	return &Project{
		ID:          getProjectIDFromSavedFilterID(sf.ID),
		Title:       sf.Title,
		Description: sf.Description,
		IsFavorite:  sf.IsFavorite,
		Created:     sf.Created,
		Updated:     sf.Updated,
		Owner:       sf.Owner,
	}
}

// Create creates a new saved filter
// @Summary Creates a new saved filter
// @Description Creates a new saved filter
// @tags filter
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Success 201 {object} models.SavedFilter "The Saved Filter"
// @Failure 403 {object} web.HTTPError "The user does not have access to that saved filter."
// @Failure 500 {object} models.Message "Internal error"
// @Router /filters [put]
func (sf *SavedFilter) Create(s *xorm.Session, auth web.Auth) error {
	sf.OwnerID = auth.GetID()
	_, err := s.Insert(sf)
	return err
}

func getSavedFilterSimpleByID(s *xorm.Session, id int64) (sf *SavedFilter, err error) {
	sf = &SavedFilter{}
	exists, err := s.
		Where("id = ?", id).
		Get(sf)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrSavedFilterDoesNotExist{SavedFilterID: id}
	}
	return
}

// ReadOne returns one saved filter
// @Summary Gets one saved filter
// @Description Returns a saved filter by its ID.
// @tags filter
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param id path int true "Filter ID"
// @Success 200 {object} models.SavedFilter "The Saved Filter"
// @Failure 403 {object} web.HTTPError "The user does not have access to that saved filter."
// @Failure 500 {object} models.Message "Internal error"
// @Router /filters/{id} [get]
func (sf *SavedFilter) ReadOne(s *xorm.Session, _ web.Auth) error {
	// s already contains almost the full saved filter from the rights check, we only need to add the user
	u, err := user.GetUserByID(s, sf.OwnerID)
	sf.Owner = u
	return err
}

// Update updates an existing filter
// @Summary Updates a saved filter
// @Description Updates a saved filter by its ID.
// @tags filter
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param id path int true "Filter ID"
// @Success 200 {object} models.SavedFilter "The Saved Filter"
// @Failure 403 {object} web.HTTPError "The user does not have access to that saved filter."
// @Failure 404 {object} web.HTTPError "The saved filter does not exist."
// @Failure 500 {object} models.Message "Internal error"
// @Router /filters/{id} [post]
func (sf *SavedFilter) Update(s *xorm.Session, _ web.Auth) error {
	origFilter, err := getSavedFilterSimpleByID(s, sf.ID)
	if err != nil {
		return err
	}

	if sf.Filters == nil {
		sf.Filters = origFilter.Filters
	}

	_, err = s.
		Where("id = ?", sf.ID).
		Cols(
			"title",
			"description",
			"filters",
			"is_favorite",
		).
		Update(sf)
	return err
}

// Delete removes a saved filter
// @Summary Removes a saved filter
// @Description Removes a saved filter by its ID.
// @tags filter
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param id path int true "Filter ID"
// @Success 200 {object} models.SavedFilter "The Saved Filter"
// @Failure 403 {object} web.HTTPError "The user does not have access to that saved filter."
// @Failure 404 {object} web.HTTPError "The saved filter does not exist."
// @Failure 500 {object} models.Message "Internal error"
// @Router /filters/{id} [delete]
func (sf *SavedFilter) Delete(s *xorm.Session, _ web.Auth) error {
	_, err := s.
		Where("id = ?", sf.ID).
		Delete(sf)
	return err
}
