// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/ui/session"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	f := form.NewSettingsForm(r)

	modifyRequest := model.UserModificationRequest{
		Username:               model.OptionalString(f.Username),
		Password:               model.OptionalString(f.Password),
		Theme:                  model.OptionalString(f.Theme),
		Language:               model.OptionalString(f.Language),
		Timezone:               model.OptionalString(f.Timezone),
		EntryDirection:         model.OptionalString(f.EntryDirection),
		EntryOrder:             model.OptionalString(f.EntryOrder),
		EntriesPerPage:         model.OptionalNumber(f.EntriesPerPage),
		CategoriesSortingOrder: model.OptionalString(f.CategoriesSortingOrder),
		DisplayMode:            model.OptionalString(f.DisplayMode),
		GestureNav:             model.OptionalString(f.GestureNav),
		DefaultReadingSpeed:    model.OptionalNumber(f.DefaultReadingSpeed),
		CJKReadingSpeed:        model.OptionalNumber(f.CJKReadingSpeed),
		DefaultHomePage:        model.OptionalString(f.DefaultHomePage),
		MediaPlaybackRate:      model.OptionalNumber(f.MediaPlaybackRate),
		BlockFilterEntryRules:  model.OptionalString(f.BlockFilterEntryRules),
		KeepFilterEntryRules:   model.OptionalString(f.KeepFilterEntryRules),
		ExternalFontHosts:      model.OptionalString(f.ExternalFontHosts),
	}

	if lerr := f.Validate(); lerr != nil {
		h.showUpdateSettingsError(w, r, func(v *View) {
			v.Set("form", f).
				Set("errorMessage", lerr.Translate(v.User().Language))
			html.OK(w, r, v.Render("settings"))
		})
		return
	}

	userID := request.UserID(r)
	lerr := validator.ValidateUserModification(r.Context(), h.store, userID,
		&modifyRequest)
	if lerr != nil {
		h.showUpdateSettingsError(w, r, func(v *View) {
			v.Set("form", f).
				Set("errorMessage", lerr.Translate(v.User().Language))
			html.OK(w, r, v.Render("settings"))
		})
		return
	}

	user := request.User(r)
	err := h.store.UpdateUser(r.Context(), f.Merge(user))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	session.New(h.store, r).
		SetLanguage(user.Language).
		SetTheme(user.Theme).
		NewFlashMessage(
			locale.NewPrinter(request.UserLanguage(r)).
				Printf("alert.prefs_saved")).
		Commit(r.Context())
	html.Redirect(w, r, route.Path(h.router, "settings"))
}

func (h *handler) showUpdateSettingsError(w http.ResponseWriter,
	r *http.Request, renderFunc func(v *View),
) {
	v := h.View(r)

	var creds []model.WebAuthnCredential
	v.Go(func(ctx context.Context) (err error) {
		creds, err = h.store.WebAuthnCredentialsByUserID(ctx, v.UserID())
		return
	})

	var timezones map[string]string
	v.Go(func(ctx context.Context) (err error) {
		timezones, err = h.store.Timezones(ctx)
		return
	})

	var webAuthnCount int
	v.Go(func(ctx context.Context) error {
		webAuthnCount = h.store.CountWebAuthnCredentialsByUserID(ctx, v.UserID())
		return nil
	})

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "settings").
		Set("readBehaviors", map[string]any{
			"NoAutoMarkAsRead":                           form.NoAutoMarkAsRead,
			"MarkAsReadOnView":                           form.MarkAsReadOnView,
			"MarkAsReadOnViewButWaitForPlayerCompletion": form.MarkAsReadOnViewButWaitForPlayerCompletion,
			"MarkAsReadOnlyOnPlayerCompletion":           form.MarkAsReadOnlyOnPlayerCompletion,
		}).
		Set("themes", model.Themes()).
		Set("languages", locale.AvailableLanguages).
		Set("timezones", timezones).
		Set("default_home_pages", model.HomePages()).
		Set("categories_sorting_options", model.CategoriesSortingOptions()).
		Set("countWebAuthnCerts", webAuthnCount).
		Set("webAuthnCerts", creds)
	renderFunc(v)
}
