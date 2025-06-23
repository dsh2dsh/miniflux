// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package validator // import "miniflux.app/v2/internal/validator"

import (
	"context"
	"strings"
	"unicode"

	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/filter"
	"miniflux.app/v2/internal/storage"
)

// ValidateUserCreationWithPassword validates user creation with a password.
func ValidateUserCreationWithPassword(ctx context.Context,
	store *storage.Storage, r *model.UserCreationRequest,
) *locale.LocalizedError {
	if r.Username == "" {
		return locale.NewLocalizedError("error.user_mandatory_fields")
	}

	if store.UserExists(ctx, r.Username) {
		return locale.NewLocalizedError("error.user_already_exists")
	}

	if err := validateUsername(r.Username); err != nil {
		return err
	}

	if err := validatePassword(r.Password); err != nil {
		return err
	}
	return nil
}

// ValidateUserModification validates user modifications.
func ValidateUserModification(ctx context.Context, store *storage.Storage,
	userID int64, r *model.UserModificationRequest,
) *locale.LocalizedError {
	if r.Username != nil {
		if *r.Username == "" {
			return locale.NewLocalizedError("error.user_mandatory_fields")
		} else if store.AnotherUserExists(ctx, userID, *r.Username) {
			return locale.NewLocalizedError("error.user_already_exists")
		}
	}

	if r.Password != nil {
		if err := validatePassword(*r.Password); err != nil {
			return err
		}
	}

	if r.Theme != nil {
		if err := validateTheme(*r.Theme); err != nil {
			return err
		}
	}

	if r.Language != nil {
		if err := validateLanguage(*r.Language); err != nil {
			return err
		}
	}

	if r.Timezone != nil {
		if err := validateTimezone(ctx, store, *r.Timezone); err != nil {
			return err
		}
	}

	if r.EntryDirection != nil {
		if err := validateEntryDirection(*r.EntryDirection); err != nil {
			return err
		}
	}

	if r.EntryOrder != nil {
		if err := ValidateEntryOrder(*r.EntryOrder); err != nil {
			return locale.NewLocalizedError("error.invalid_entry_order")
		}
	}

	if r.EntriesPerPage != nil {
		if err := validateEntriesPerPage(*r.EntriesPerPage); err != nil {
			return err
		}
	}

	if r.CategoriesSortingOrder != nil {
		if err := validateCategoriesSortingOrder(*r.CategoriesSortingOrder); err != nil {
			return err
		}
	}

	if r.DisplayMode != nil {
		if err := validateDisplayMode(*r.DisplayMode); err != nil {
			return err
		}
	}

	if r.GestureNav != nil {
		if err := validateGestureNav(*r.GestureNav); err != nil {
			return err
		}
	}

	if r.DefaultReadingSpeed != nil {
		if err := validateReadingSpeed(*r.DefaultReadingSpeed); err != nil {
			return err
		}
	}

	if r.CJKReadingSpeed != nil {
		if err := validateReadingSpeed(*r.CJKReadingSpeed); err != nil {
			return err
		}
	}

	if r.DefaultHomePage != nil {
		if err := validateDefaultHomePage(*r.DefaultHomePage); err != nil {
			return err
		}
	}

	if r.MediaPlaybackRate != nil {
		if err := validateMediaPlaybackRate(*r.MediaPlaybackRate); err != nil {
			return err
		}
	}

	if s := model.OptionalValue(r.BlockFilterEntryRules); s != "" {
		if _, err := filter.New(s); err != nil {
			return locale.NewLocalizedError(
				"The block list rule is invalid: " + err.Error())
		}
	}

	if s := model.OptionalValue(r.KeepFilterEntryRules); s != "" {
		if _, err := filter.New(s); err != nil {
			return locale.NewLocalizedError(
				"The keep list rule is invalid: " + err.Error())
		}
	}

	if r.ExternalFontHosts != nil {
		if !IsValidDomainList(*r.ExternalFontHosts) {
			return locale.NewLocalizedError("error.settings_invalid_domain_list")
		}
	}

	return nil
}

func validateReadingSpeed(readingSpeed int) *locale.LocalizedError {
	if readingSpeed <= 0 {
		return locale.NewLocalizedError("error.settings_reading_speed_is_positive")
	}
	return nil
}

func validatePassword(password string) *locale.LocalizedError {
	if len(password) < 6 {
		return locale.NewLocalizedError("error.password_min_length")
	}
	return nil
}

// validateUsername return an error if the `username` argument contains
// a character that isn't alphanumerical nor `_` and `-`.
//
// Note: this validation should not be applied to previously created usernames,
// and cannot be applied to Google/OIDC accounts creation because the email
// address is used for the username field.
func validateUsername(username string) *locale.LocalizedError {
	if strings.ContainsFunc(username, func(r rune) bool {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return false
		}
		if r == '_' || r == '-' || r == '@' || r == '.' {
			return false
		}
		return true
	}) {
		return locale.NewLocalizedError("error.invalid_username")
	}
	return nil
}

func validateTheme(theme string) *locale.LocalizedError {
	themes := model.Themes()
	if _, found := themes[theme]; !found {
		return locale.NewLocalizedError("error.invalid_theme")
	}
	return nil
}

func validateLanguage(language string) *locale.LocalizedError {
	languages := locale.AvailableLanguages
	if _, found := languages[language]; !found {
		return locale.NewLocalizedError("error.invalid_language")
	}
	return nil
}

func validateTimezone(ctx context.Context, store *storage.Storage,
	timezone string,
) *locale.LocalizedError {
	timezones, err := store.Timezones(ctx)
	if err != nil {
		return locale.NewLocalizedError(err.Error())
	}

	if _, found := timezones[timezone]; !found {
		return locale.NewLocalizedError("error.invalid_timezone")
	}
	return nil
}

func validateEntryDirection(direction string) *locale.LocalizedError {
	if direction != "asc" && direction != "desc" {
		return locale.NewLocalizedError("error.invalid_entry_direction")
	}
	return nil
}

func validateEntriesPerPage(entriesPerPage int) *locale.LocalizedError {
	if entriesPerPage < 1 {
		return locale.NewLocalizedError("error.entries_per_page_invalid")
	}
	return nil
}

func validateCategoriesSortingOrder(order string) *locale.LocalizedError {
	if order != "alphabetical" && order != "unread_count" {
		return locale.NewLocalizedError("error.invalid_categories_sorting_order")
	}
	return nil
}

func validateDisplayMode(displayMode string) *locale.LocalizedError {
	if displayMode != "fullscreen" && displayMode != "standalone" && displayMode != "minimal-ui" && displayMode != "browser" {
		return locale.NewLocalizedError("error.invalid_display_mode")
	}
	return nil
}

func validateGestureNav(gestureNav string) *locale.LocalizedError {
	if gestureNav != "none" && gestureNav != "tap" && gestureNav != "swipe" {
		return locale.NewLocalizedError("error.invalid_gesture_nav")
	}
	return nil
}

func validateDefaultHomePage(defaultHomePage string) *locale.LocalizedError {
	defaultHomePages := model.HomePages()
	if _, found := defaultHomePages[defaultHomePage]; !found {
		return locale.NewLocalizedError("error.invalid_default_home_page")
	}
	return nil
}

func validateMediaPlaybackRate(mediaPlaybackRate float64) *locale.LocalizedError {
	if mediaPlaybackRate < 0.25 || mediaPlaybackRate > 4 {
		return locale.NewLocalizedError("error.settings_media_playback_rate_range")
	}
	return nil
}
