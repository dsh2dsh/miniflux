// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"

	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
)

func (h *handler) showSettingsPage(w http.ResponseWriter, r *http.Request) {
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

	user := v.User()
	settingsForm := form.SettingsForm{
		Username:               user.Username,
		Theme:                  user.Theme,
		Language:               user.Language,
		Timezone:               user.Timezone,
		EntryDirection:         user.EntryDirection,
		EntryOrder:             user.EntryOrder,
		EntriesPerPage:         user.EntriesPerPage,
		KeyboardShortcuts:      user.KeyboardShortcuts,
		ShowReadingTime:        user.ShowReadingTime,
		CustomCSS:              user.Stylesheet,
		CustomJS:               user.CustomJS,
		ExternalFontHosts:      user.ExternalFontHosts,
		EntrySwipe:             user.EntrySwipe,
		GestureNav:             user.GestureNav,
		DisplayMode:            user.DisplayMode,
		DefaultReadingSpeed:    user.DefaultReadingSpeed,
		CJKReadingSpeed:        user.CJKReadingSpeed,
		DefaultHomePage:        user.DefaultHomePage,
		CategoriesSortingOrder: user.CategoriesSortingOrder,
		MarkReadBehavior: form.MarkAsReadBehavior(user.MarkReadOnView,
			user.MarkReadOnMediaPlayerCompletion),
		MediaPlaybackRate:     user.MediaPlaybackRate,
		BlockFilterEntryRules: user.BlockFilterEntryRules,
		KeepFilterEntryRules:  user.KeepFilterEntryRules,
	}

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
	html.OK(w, r, v.Render("settings"))
}
