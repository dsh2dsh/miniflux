// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) updateSettings(w http.ResponseWriter, r *http.Request) {
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

	settingsForm := form.NewSettingsForm(r)
	v.Set("menu", "settings").
		Set("form", settingsForm).
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

	// Sanitize the end of the block & Keep rules
	cleanEnd := regexp.MustCompile(`(?m)\r\n\s*$`)
	settingsForm.BlockFilterEntryRules = cleanEnd.ReplaceAllLiteralString(
		settingsForm.BlockFilterEntryRules, "")
	settingsForm.KeepFilterEntryRules = cleanEnd.ReplaceAllLiteralString(
		settingsForm.KeepFilterEntryRules, "")

	// Clean carriage returns for Windows environments
	settingsForm.BlockFilterEntryRules = strings.ReplaceAll(
		settingsForm.BlockFilterEntryRules, "\r\n", "\n")
	settingsForm.KeepFilterEntryRules = strings.ReplaceAll(
		settingsForm.KeepFilterEntryRules, "\r\n", "\n")

	if lerr := settingsForm.Validate(); lerr != nil {
		v.Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("settings"))
		return
	}

	userModificationRequest := &model.UserModificationRequest{
		Username:               model.OptionalString(settingsForm.Username),
		Password:               model.OptionalString(settingsForm.Password),
		Theme:                  model.OptionalString(settingsForm.Theme),
		Language:               model.OptionalString(settingsForm.Language),
		Timezone:               model.OptionalString(settingsForm.Timezone),
		EntryDirection:         model.OptionalString(settingsForm.EntryDirection),
		EntryOrder:             model.OptionalString(settingsForm.EntryOrder),
		EntriesPerPage:         model.OptionalNumber(settingsForm.EntriesPerPage),
		CategoriesSortingOrder: model.OptionalString(settingsForm.CategoriesSortingOrder),
		DisplayMode:            model.OptionalString(settingsForm.DisplayMode),
		GestureNav:             model.OptionalString(settingsForm.GestureNav),
		DefaultReadingSpeed:    model.OptionalNumber(settingsForm.DefaultReadingSpeed),
		CJKReadingSpeed:        model.OptionalNumber(settingsForm.CJKReadingSpeed),
		DefaultHomePage:        model.OptionalString(settingsForm.DefaultHomePage),
		MediaPlaybackRate:      model.OptionalNumber(settingsForm.MediaPlaybackRate),
		BlockFilterEntryRules:  model.OptionalString(settingsForm.BlockFilterEntryRules),
		KeepFilterEntryRules:   model.OptionalString(settingsForm.KeepFilterEntryRules),
		ExternalFontHosts:      model.OptionalString(settingsForm.ExternalFontHosts),
	}

	lerr := validator.ValidateUserModification(r.Context(),
		h.store, v.UserID(), userModificationRequest)
	if lerr != nil {
		v.Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("settings"))
		return
	}

	err := h.store.UpdateUser(r.Context(), settingsForm.Merge(v.User()))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	sess := v.Session()
	sess.SetLanguage(r.Context(), v.User().Language)
	sess.SetTheme(r.Context(), v.User().Theme)
	sess.NewFlashMessage(r.Context(),
		locale.NewPrinter(request.UserLanguage(r)).Printf("alert.prefs_saved"))
	html.Redirect(w, r, route.Path(h.router, "settings"))
}
