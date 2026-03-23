// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	json_parser "encoding/json"
	"errors"
	"net/http"
	"strconv"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) currentUser(w http.ResponseWriter, r *http.Request,
) (*model.User, error) {
	return request.User(r), nil
}

func (h *handler) createUser(w http.ResponseWriter, r *http.Request,
) (*model.User, error) {
	if !request.IsAdminUser(r) {
		return nil, response.ErrForbidden
	}

	var createRequest model.UserCreationRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&createRequest); err != nil {
		return nil, response.WrapBadRequest(err)
	}

	ctx := r.Context()
	lerr := validator.ValidateUserCreationWithPassword(ctx, h.store,
		&createRequest)
	if lerr != nil {
		return nil, response.WrapBadRequest(lerr.Error())
	}

	user, err := h.store.CreateUser(ctx, &createRequest)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (h *handler) updateUser(w http.ResponseWriter, r *http.Request,
) (*model.User, error) {
	id := request.RouteInt64Param(r, "userID")
	var m model.UserModificationRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&m); err != nil {
		return nil, response.WrapBadRequest(err)
	}

	ctx := r.Context()
	user, err := h.store.UserByID(ctx, id)
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, response.ErrNotFound
	}

	if !request.IsAdminUser(r) {
		if user.ID != request.UserID(r) {
			return nil, response.ErrForbidden
		}

		if m.IsAdmin != nil && *m.IsAdmin {
			return nil, response.WrapBadRequest(errors.New(
				"only administrators can change permissions of standard users"))
		}
	}

	lerr := validator.ValidateUserModification(ctx, h.store, user.ID, &m)
	if lerr != nil {
		return nil, response.WrapBadRequest(lerr.Error())
	}

	m.Patch(user)
	if err = h.store.UpdateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (h *handler) markUserAsRead(w http.ResponseWriter, r *http.Request) error {
	id := request.RouteInt64Param(r, "userID")
	if id != request.UserID(r) {
		return response.ErrForbidden
	}

	user := request.User(r)
	if err := h.store.MarkAllAsRead(r.Context(), user.ID); err != nil {
		return err
	}
	return nil
}

func (h *handler) getIntegrationsStatus(w http.ResponseWriter, r *http.Request,
) (*integrationsStatusResponse, error) {
	user := request.User(r)
	return &integrationsStatusResponse{HasIntegrations: user.HasSaveEntry()}, nil
}

func (h *handler) users(w http.ResponseWriter, r *http.Request,
) (model.Users, error) {
	if !request.IsAdminUser(r) {
		return nil, response.ErrForbidden
	}

	users, err := h.store.Users(r.Context())
	if err != nil {
		return nil, err
	}

	users.UseTimezone(request.UserTimezone(r))
	return users, nil
}

func (h *handler) userByID(w http.ResponseWriter, r *http.Request,
) (*model.User, error) {
	if !request.IsAdminUser(r) {
		return nil, response.ErrForbidden
	}

	username := request.RouteStringParam(r, "userID")
	if username == "" {
		return nil, response.ErrNotFound
	}

	userID, err := strconv.ParseInt(username, 10, 64)
	if err != nil {
		return h.userByUsername(w, r, username)
	}

	user, err := h.store.UserByID(r.Context(), userID)
	if err != nil {
		return nil, response.WrapBadRequest(errors.New(
			"unable to fetch this user from the database"))
	} else if user == nil {
		return nil, response.ErrNotFound
	}

	user.UseTimezone(request.UserTimezone(r))
	return user, nil
}

func (h *handler) userByUsername(_ http.ResponseWriter, r *http.Request,
	username string,
) (*model.User, error) {
	user, err := h.store.UserByUsername(r.Context(), username)
	if err != nil {
		return nil, response.WrapBadRequest(errors.New(
			"unable to fetch this user from the database"))
	} else if user == nil {
		return nil, response.ErrNotFound
	}

	user.UseTimezone(request.UserTimezone(r))
	return user, nil
}

func (h *handler) removeUser(w http.ResponseWriter, r *http.Request) error {
	userID := request.RouteInt64Param(r, "userID")
	if !request.IsAdminUser(r) {
		return response.ErrForbidden
	} else if userID == request.UserID(r) {
		return response.WrapBadRequest(errors.New("you cannot remove yourself"))
	}

	affected, err := h.store.RemoveUser(r.Context(), userID)
	if err != nil {
		return err
	} else if !affected {
		return response.ErrNotFound
	}
	return nil
}
