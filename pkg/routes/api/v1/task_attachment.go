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

package v1

import (
	"net/http"

	"code.vikunja.io/api/pkg/db"

	"code.vikunja.io/api/pkg/models"
	auth2 "code.vikunja.io/api/pkg/modules/auth"
	"code.vikunja.io/web/handler"
	"github.com/labstack/echo/v4"
)

// UploadTaskAttachment handles everything needed for the upload of a task attachment
// @Summary Upload a task attachment
// @Description Upload a task attachment. You can pass multiple files with the files form param.
// @tags task
// @Accept mpfd
// @Produce json
// @Param id path int true "Task ID"
// @Param files formData string true "The file, as multipart form file. You can pass multiple."
// @Security JWTKeyAuth
// @Success 200 {object} models.Message "Attachments were uploaded successfully."
// @Failure 403 {object} models.Message "No access to the task."
// @Failure 404 {object} models.Message "The task does not exist."
// @Failure 500 {object} models.Message "Internal error"
// @Router /tasks/{id}/attachments [put]
func UploadTaskAttachment(c echo.Context) error {

	var taskAttachment models.TaskAttachment
	if err := c.Bind(&taskAttachment); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "No task ID provided")
	}

	// Rights check
	auth, err := auth2.GetAuthFromClaims(c)
	if err != nil {
		return handler.HandleHTTPError(err, c)
	}

	s := db.NewSession()
	defer s.Close()

	can, err := taskAttachment.CanCreate(s, auth)
	if err != nil {
		_ = s.Rollback()
		return handler.HandleHTTPError(err, c)
	}
	if !can {
		return echo.ErrForbidden
	}

	// Multipart form
	form, err := c.MultipartForm()
	if err != nil {
		_ = s.Rollback()
		return handler.HandleHTTPError(err, c)
	}

	type result struct {
		Errors  []*echo.HTTPError        `json:"errors"`
		Success []*models.TaskAttachment `json:"success"`
	}
	r := &result{}
	fileHeaders := form.File["files"]
	for _, file := range fileHeaders {
		// We create a new attachment object here to have a clean start
		ta := &models.TaskAttachment{
			TaskID: taskAttachment.TaskID,
		}

		f, err := file.Open()
		if err != nil {
			r.Errors = append(r.Errors, handler.HandleHTTPError(err, c))
			continue
		}
		defer f.Close()

		err = ta.NewAttachment(s, f, file.Filename, uint64(file.Size), auth)
		if err != nil {
			r.Errors = append(r.Errors, handler.HandleHTTPError(err, c))
			continue
		}
		r.Success = append(r.Success, ta)
	}

	if err := s.Commit(); err != nil {
		_ = s.Rollback()
		return handler.HandleHTTPError(err, c)
	}

	return c.JSON(http.StatusOK, r)
}

// GetTaskAttachment returns a task attachment to download for the user
// @Summary Get one attachment.
// @Description Get one attachment for download. **Returns json on error.**
// @tags task
// @Produce octet-stream
// @Param id path int true "Task ID"
// @Param attachmentID path int true "Attachment ID"
// @Security JWTKeyAuth
// @Success 200 {file} blob "The attachment file."
// @Failure 403 {object} models.Message "No access to this task."
// @Failure 404 {object} models.Message "The task does not exist."
// @Failure 500 {object} models.Message "Internal error"
// @Router /tasks/{id}/attachments/{attachmentID} [get]
func GetTaskAttachment(c echo.Context) error {

	var taskAttachment models.TaskAttachment
	if err := c.Bind(&taskAttachment); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "No task ID provided")
	}

	// Rights check
	auth, err := auth2.GetAuthFromClaims(c)
	if err != nil {
		return handler.HandleHTTPError(err, c)
	}

	s := db.NewSession()
	defer s.Close()

	can, _, err := taskAttachment.CanRead(s, auth)
	if err != nil {
		_ = s.Rollback()
		return handler.HandleHTTPError(err, c)
	}
	if !can {
		return echo.ErrForbidden
	}

	// Get the attachment incl file
	err = taskAttachment.ReadOne(s, auth)
	if err != nil {
		_ = s.Rollback()
		return handler.HandleHTTPError(err, c)
	}

	// Open an send the file to the client
	err = taskAttachment.File.LoadFileByID()
	if err != nil {
		_ = s.Rollback()
		return handler.HandleHTTPError(err, c)
	}

	if err := s.Commit(); err != nil {
		_ = s.Rollback()
		return handler.HandleHTTPError(err, c)
	}

	http.ServeContent(c.Response(), c.Request(), taskAttachment.File.Name, taskAttachment.File.Created, taskAttachment.File.File)
	return nil
}
