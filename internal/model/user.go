// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/timezone"
)

// User represents a user in the system.
type User struct {
	ID                              int64      `json:"id" db:"id"`
	Username                        string     `json:"username" db:"username"`
	Password                        string     `json:"-" db:"password"`
	IsAdmin                         bool       `json:"is_admin" db:"is_admin"`
	Theme                           string     `json:"theme" db:"theme"`
	Language                        string     `json:"language" db:"language"`
	Timezone                        string     `json:"timezone" db:"timezone"`
	EntryDirection                  string     `json:"entry_sorting_direction" db:"entry_direction"`
	EntryOrder                      string     `json:"entry_sorting_order" db:"entry_order"`
	Stylesheet                      string     `json:"stylesheet" db:"stylesheet"`
	CustomJS                        string     `json:"custom_js" db:"custom_js"`
	ExternalFontHosts               string     `json:"external_font_hosts" db:"external_font_hosts"`
	GoogleID                        string     `json:"google_id" db:"google_id"`
	OpenIDConnectID                 string     `json:"openid_connect_id" db:"openid_connect_id"`
	EntriesPerPage                  int        `json:"entries_per_page" db:"entries_per_page"`
	KeyboardShortcuts               bool       `json:"keyboard_shortcuts" db:"keyboard_shortcuts"`
	ShowReadingTime                 bool       `json:"show_reading_time" db:"show_reading_time"`
	EntrySwipe                      bool       `json:"entry_swipe" db:"entry_swipe"`
	GestureNav                      string     `json:"gesture_nav" db:"gesture_nav"`
	LastLoginAt                     *time.Time `json:"last_login_at" db:"last_login_at"`
	DisplayMode                     string     `json:"display_mode" db:"display_mode"`
	DefaultReadingSpeed             int        `json:"default_reading_speed" db:"default_reading_speed"`
	CJKReadingSpeed                 int        `json:"cjk_reading_speed" db:"cjk_reading_speed"`
	DefaultHomePage                 string     `json:"default_home_page" db:"default_home_page"`
	CategoriesSortingOrder          string     `json:"categories_sorting_order" db:"categories_sorting_order"`
	MarkReadOnView                  bool       `json:"mark_read_on_view" db:"mark_read_on_view"`
	MarkReadOnMediaPlayerCompletion bool       `json:"mark_read_on_media_player_completion"`
	MediaPlaybackRate               float64    `json:"media_playback_rate" db:"media_playback_rate"`
	BlockFilterEntryRules           string     `json:"block_filter_entry_rules" db:"block_filter_entry_rules"`
	KeepFilterEntryRules            string     `json:"keep_filter_entry_rules" db:"keep_filter_entry_rules"`
	Extra                           UserExtra  `json:"extra,omitzero" db:"extra"`
}

type UserExtra struct {
	AlwaysOpenExternalLinks bool        `json:"always_open_external_links,omitempty"`
	Integration             Integration `json:"integration,omitzero"`
	OpenExternalLinkSameTab bool        `json:"open_external_link_same_tab,omitempty"`
}

// UserCreationRequest represents the request to create a user.
type UserCreationRequest struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	IsAdmin         bool   `json:"is_admin"`
	GoogleID        string `json:"google_id"`
	OpenIDConnectID string `json:"openid_connect_id"`
}

// UserModificationRequest represents the request to update a user.
type UserModificationRequest struct {
	Username                        *string  `json:"username"`
	Password                        *string  `json:"password"`
	Theme                           *string  `json:"theme"`
	Language                        *string  `json:"language"`
	Timezone                        *string  `json:"timezone"`
	EntryDirection                  *string  `json:"entry_sorting_direction"`
	EntryOrder                      *string  `json:"entry_sorting_order"`
	Stylesheet                      *string  `json:"stylesheet"`
	CustomJS                        *string  `json:"custom_js"`
	ExternalFontHosts               *string  `json:"external_font_hosts"`
	GoogleID                        *string  `json:"google_id"`
	OpenIDConnectID                 *string  `json:"openid_connect_id"`
	EntriesPerPage                  *int     `json:"entries_per_page"`
	IsAdmin                         *bool    `json:"is_admin"`
	KeyboardShortcuts               *bool    `json:"keyboard_shortcuts"`
	ShowReadingTime                 *bool    `json:"show_reading_time"`
	EntrySwipe                      *bool    `json:"entry_swipe"`
	GestureNav                      *string  `json:"gesture_nav"`
	DisplayMode                     *string  `json:"display_mode"`
	DefaultReadingSpeed             *int     `json:"default_reading_speed"`
	CJKReadingSpeed                 *int     `json:"cjk_reading_speed"`
	DefaultHomePage                 *string  `json:"default_home_page"`
	CategoriesSortingOrder          *string  `json:"categories_sorting_order"`
	MarkReadOnView                  *bool    `json:"mark_read_on_view"`
	MarkReadOnMediaPlayerCompletion *bool    `json:"mark_read_on_media_player_completion"`
	MediaPlaybackRate               *float64 `json:"media_playback_rate"`
	BlockFilterEntryRules           *string  `json:"block_filter_entry_rules"`
	KeepFilterEntryRules            *string  `json:"keep_filter_entry_rules"`
	AlwaysOpenExternalLinks         *bool    `json:"always_open_external_links,omitempty"`
	OpenExternalLinkSameTab         *bool    `json:"open_external_link_same_tab,omitempty"`
}

// Patch updates the User object with the modification request.
func (u *UserModificationRequest) Patch(user *User) {
	if u.Username != nil {
		user.Username = *u.Username
	}

	if u.Password != nil {
		user.Password = *u.Password
	}

	if u.IsAdmin != nil {
		user.IsAdmin = *u.IsAdmin
	}

	if u.Theme != nil {
		user.Theme = *u.Theme
	}

	if u.Language != nil {
		user.Language = *u.Language
	}

	if u.Timezone != nil {
		user.Timezone = *u.Timezone
	}

	if u.EntryDirection != nil {
		user.EntryDirection = *u.EntryDirection
	}

	if u.EntryOrder != nil {
		user.EntryOrder = *u.EntryOrder
	}

	if u.Stylesheet != nil {
		user.Stylesheet = *u.Stylesheet
	}

	if u.CustomJS != nil {
		user.CustomJS = *u.CustomJS
	}

	if u.ExternalFontHosts != nil {
		user.ExternalFontHosts = *u.ExternalFontHosts
	}

	if u.GoogleID != nil {
		user.GoogleID = *u.GoogleID
	}

	if u.OpenIDConnectID != nil {
		user.OpenIDConnectID = *u.OpenIDConnectID
	}

	if u.EntriesPerPage != nil {
		user.EntriesPerPage = *u.EntriesPerPage
	}

	if u.KeyboardShortcuts != nil {
		user.KeyboardShortcuts = *u.KeyboardShortcuts
	}

	if u.ShowReadingTime != nil {
		user.ShowReadingTime = *u.ShowReadingTime
	}

	if u.EntrySwipe != nil {
		user.EntrySwipe = *u.EntrySwipe
	}

	if u.GestureNav != nil {
		user.GestureNav = *u.GestureNav
	}

	if u.DisplayMode != nil {
		user.DisplayMode = *u.DisplayMode
	}

	if u.DefaultReadingSpeed != nil {
		user.DefaultReadingSpeed = *u.DefaultReadingSpeed
	}

	if u.CJKReadingSpeed != nil {
		user.CJKReadingSpeed = *u.CJKReadingSpeed
	}

	if u.DefaultHomePage != nil {
		user.DefaultHomePage = *u.DefaultHomePage
	}

	if u.CategoriesSortingOrder != nil {
		user.CategoriesSortingOrder = *u.CategoriesSortingOrder
	}

	if u.MarkReadOnView != nil {
		user.MarkReadOnView = *u.MarkReadOnView
	}

	if u.MarkReadOnMediaPlayerCompletion != nil {
		user.MarkReadOnMediaPlayerCompletion = *u.MarkReadOnMediaPlayerCompletion
	}

	if u.MediaPlaybackRate != nil {
		user.MediaPlaybackRate = *u.MediaPlaybackRate
	}

	if u.BlockFilterEntryRules != nil {
		user.BlockFilterEntryRules = *u.BlockFilterEntryRules
	}

	if u.KeepFilterEntryRules != nil {
		user.KeepFilterEntryRules = *u.KeepFilterEntryRules
	}

	if u.AlwaysOpenExternalLinks != nil {
		user.Extra.AlwaysOpenExternalLinks = *u.AlwaysOpenExternalLinks
	}

	if u.OpenExternalLinkSameTab != nil {
		user.Extra.OpenExternalLinkSameTab = *u.OpenExternalLinkSameTab
	}
}

func (u *User) String() string {
	return fmt.Sprintf("#%d - %s (admin=%v)", u.ID, u.Username, u.IsAdmin)
}

// UseTimezone converts last login date to the given timezone.
func (u *User) UseTimezone(tz string) {
	if u.LastLoginAt != nil {
		timezone.Convert(tz, u.LastLoginAt)
	}
}

func (u *User) AlwaysOpenExternalLinks() bool {
	return u.Extra.AlwaysOpenExternalLinks
}

func (u *User) Integration() *Integration { return &u.Extra.Integration }

func (u *User) OpenExternalLinkSameTab() bool {
	return u.Extra.OpenExternalLinkSameTab
}

func (u *User) TargetBlank() template.HTMLAttr {
	if u.OpenExternalLinkSameTab() {
		return ""
	}
	return `target="_blank"`
}

func (u *User) HasSaveEntry() bool {
	i := u.Integration()
	return i.AppriseEnabled ||
		i.ArchiveorgEnabled ||
		i.BetulaEnabled ||
		i.CuboxEnabled ||
		i.DiscordEnabled ||
		i.EspialEnabled ||
		i.InstapaperEnabled ||
		i.KarakeepEnabled ||
		i.LinkAceEnabled ||
		i.LinkdingEnabled ||
		i.LinkwardenEnabled ||
		i.NotionEnabled ||
		i.NunuxKeeperEnabled ||
		i.OmnivoreEnabled ||
		i.PinboardEnabled ||
		i.RaindropEnabled ||
		i.ReadeckEnabled ||
		i.ReadwiseEnabled ||
		i.ShaarliEnabled ||
		i.ShioriEnabled ||
		i.SlackEnabled ||
		i.WallabagEnabled ||
		i.WebhookEnabled
}

func (u *User) Operator() bool {
	return u.IsAdmin || config.Opts.Operator(u.Username)
}

func (u *User) HasStylesheet() bool {
	return strings.TrimSpace(u.Stylesheet) != ""
}

func (u *User) StylesheetHash() string {
	return crypto.HashFromString(u.Stylesheet)
}

func (u *User) HasJavascript() bool {
	return strings.TrimSpace(u.CustomJS) != ""
}

func (u *User) JavascriptHash() string {
	return crypto.HashFromString(u.CustomJS)
}

// Users represents a list of users.
type Users []*User

// UseTimezone converts last login timestamp of all users to the given timezone.
func (u Users) UseTimezone(tz string) {
	for _, user := range u {
		user.UseTimezone(tz)
	}
}
