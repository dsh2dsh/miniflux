// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	json_parser "encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/validator"
)

var cleanEnd = regexp.MustCompile(`(?m)\r\n\s*$`)

func (h *handler) currentUser(w http.ResponseWriter, r *http.Request) {
	json.OK(w, r, request.User(r))
}

func (h *handler) createUser(w http.ResponseWriter, r *http.Request) {
	if !request.IsAdminUser(r) {
		json.Forbidden(w, r)
		return
	}

	var createRequest model.UserCreationRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&createRequest); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	ctx := r.Context()
	lerr := validator.ValidateUserCreationWithPassword(ctx, h.store,
		&createRequest)
	if lerr != nil {
		json.BadRequest(w, r, lerr.Error())
		return
	}

	user, err := h.store.CreateUser(ctx, &createRequest)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.Created(w, r, user)
}

func (h *handler) updateUser(w http.ResponseWriter, r *http.Request) {
	id := request.RouteInt64Param(r, "userID")
	var m model.UserModificationRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&m); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	ctx := r.Context()
	user, err := h.store.UserByID(ctx, id)
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if user == nil {
		json.NotFound(w, r)
		return
	}

	if !request.IsAdminUser(r) {
		if user.ID != request.UserID(r) {
			json.Forbidden(w, r)
			return
		}

		if m.IsAdmin != nil && *m.IsAdmin {
			json.BadRequest(w, r, errors.New(
				"only administrators can change permissions of standard users"))
			return
		}
	}

	if m.BlockFilterEntryRules != nil {
		*m.BlockFilterEntryRules = cleanEnd.ReplaceAllLiteralString(
			*m.BlockFilterEntryRules, "")
		// Clean carriage returns for Windows environments
		*m.BlockFilterEntryRules = strings.ReplaceAll(*m.BlockFilterEntryRules,
			"\r\n", "\n")
	}

	if m.KeepFilterEntryRules != nil {
		*m.KeepFilterEntryRules = cleanEnd.ReplaceAllLiteralString(
			*m.KeepFilterEntryRules, "")
		// Clean carriage returns for Windows environments
		*m.KeepFilterEntryRules = strings.ReplaceAll(*m.KeepFilterEntryRules,
			"\r\n", "\n")
	}

	lerr := validator.ValidateUserModification(ctx, h.store, user.ID, &m)
	if lerr != nil {
		json.BadRequest(w, r, lerr.Error())
		return
	}

	m.Patch(user)
	if err = h.store.UpdateUser(ctx, user); err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.Created(w, r, user)
}

func (h *handler) markUserAsRead(w http.ResponseWriter, r *http.Request) {
	id := request.RouteInt64Param(r, "userID")
	if id != request.UserID(r) {
		json.Forbidden(w, r)
		return
	}

	if err := h.store.MarkAllAsRead(r.Context(), id); err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.NoContent(w, r)
}

func (h *handler) getIntegrationsStatus(w http.ResponseWriter, r *http.Request,
) {
	user := request.User(r)
	response := struct {
		HasIntegrations bool `json:"has_integrations"`
	}{
		HasIntegrations: user.HasSaveEntry(),
	}
	json.OK(w, r, &response)
}

func (h *handler) users(w http.ResponseWriter, r *http.Request) {
	if !request.IsAdminUser(r) {
		json.Forbidden(w, r)
		return
	}

	users, err := h.store.Users(r.Context())
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	users.UseTimezone(request.UserTimezone(r))
	json.OK(w, r, users)
}

func (h *handler) userByID(w http.ResponseWriter, r *http.Request) {
	if !request.IsAdminUser(r) {
		json.Forbidden(w, r)
		return
	}

	userID := request.RouteInt64Param(r, "userID")
	user, err := h.store.UserByID(r.Context(), userID)
	if err != nil {
		json.BadRequest(w, r, errors.New(
			"unable to fetch this user from the database"))
		return
	} else if user == nil {
		json.NotFound(w, r)
		return
	}

	user.UseTimezone(request.UserTimezone(r))
	json.OK(w, r, user)
}

func (h *handler) userByUsername(w http.ResponseWriter, r *http.Request) {
	if !request.IsAdminUser(r) {
		json.Forbidden(w, r)
		return
	}

	username := request.RouteStringParam(r, "username")
	user, err := h.store.UserByUsername(r.Context(), username)
	if err != nil {
		json.BadRequest(w, r, errors.New(
			"unable to fetch this user from the database"))
		return
	} else if user == nil {
		json.NotFound(w, r)
		return
	}
	json.OK(w, r, user)
}

func (h *handler) removeUser(w http.ResponseWriter, r *http.Request) {
	userID := request.RouteInt64Param(r, "userID")
	if !request.IsAdminUser(r) {
		json.Forbidden(w, r)
		return
	} else if userID == request.UserID(r) {
		json.BadRequest(w, r, errors.New("you cannot remove yourself"))
		return
	}

	affected, err := h.store.RemoveUser(r.Context(), userID)
	if err != nil {
		json.ServerError(w, r, err)
	} else if !affected {
		json.NotFound(w, r)
		return
	}
	json.NoContent(w, r)
}
