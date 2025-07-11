// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/ui/form"
)

func (h *handler) showIntegrationPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	i := v.User().Integration()
	f := form.IntegrationForm{
		PinboardEnabled:                  i.PinboardEnabled,
		PinboardToken:                    i.PinboardToken,
		PinboardTags:                     i.PinboardTags,
		PinboardMarkAsUnread:             i.PinboardMarkAsUnread,
		InstapaperEnabled:                i.InstapaperEnabled,
		InstapaperUsername:               i.InstapaperUsername,
		InstapaperPassword:               i.InstapaperPassword,
		FeverEnabled:                     i.FeverEnabled,
		GoogleReaderEnabled:              i.GoogleReaderEnabled,
		WallabagEnabled:                  i.WallabagEnabled,
		WallabagOnlyURL:                  i.WallabagOnlyURL,
		WallabagURL:                      i.WallabagURL,
		WallabagClientID:                 i.WallabagClientID,
		WallabagClientSecret:             i.WallabagClientSecret,
		WallabagUsername:                 i.WallabagUsername,
		WallabagPassword:                 i.WallabagPassword,
		NotionEnabled:                    i.NotionEnabled,
		NotionPageID:                     i.NotionPageID,
		NotionToken:                      i.NotionToken,
		NunuxKeeperEnabled:               i.NunuxKeeperEnabled,
		NunuxKeeperURL:                   i.NunuxKeeperURL,
		NunuxKeeperAPIKey:                i.NunuxKeeperAPIKey,
		EspialEnabled:                    i.EspialEnabled,
		EspialURL:                        i.EspialURL,
		EspialAPIKey:                     i.EspialAPIKey,
		EspialTags:                       i.EspialTags,
		ReadwiseEnabled:                  i.ReadwiseEnabled,
		ReadwiseAPIKey:                   i.ReadwiseAPIKey,
		TelegramBotEnabled:               i.TelegramBotEnabled,
		TelegramBotToken:                 i.TelegramBotToken,
		TelegramBotChatID:                i.TelegramBotChatID,
		TelegramBotTopicID:               i.TelegramBotTopicID,
		TelegramBotDisableWebPagePreview: i.TelegramBotDisableWebPagePreview,
		TelegramBotDisableNotification:   i.TelegramBotDisableNotification,
		TelegramBotDisableButtons:        i.TelegramBotDisableButtons,
		LinkAceEnabled:                   i.LinkAceEnabled,
		LinkAceURL:                       i.LinkAceURL,
		LinkAceAPIKey:                    i.LinkAceAPIKey,
		LinkAceTags:                      i.LinkAceTags,
		LinkAcePrivate:                   i.LinkAcePrivate,
		LinkAceCheckDisabled:             i.LinkAceCheckDisabled,
		LinkdingEnabled:                  i.LinkdingEnabled,
		LinkdingURL:                      i.LinkdingURL,
		LinkdingAPIKey:                   i.LinkdingAPIKey,
		LinkdingTags:                     i.LinkdingTags,
		LinkdingMarkAsUnread:             i.LinkdingMarkAsUnread,
		LinkwardenEnabled:                i.LinkwardenEnabled,
		LinkwardenURL:                    i.LinkwardenURL,
		LinkwardenAPIKey:                 i.LinkwardenAPIKey,
		MatrixBotEnabled:                 i.MatrixBotEnabled,
		MatrixBotUser:                    i.MatrixBotUser,
		MatrixBotPassword:                i.MatrixBotPassword,
		MatrixBotURL:                     i.MatrixBotURL,
		MatrixBotChatID:                  i.MatrixBotChatID,
		AppriseEnabled:                   i.AppriseEnabled,
		AppriseURL:                       i.AppriseURL,
		AppriseServicesURL:               i.AppriseServicesURL,
		ReadeckEnabled:                   i.ReadeckEnabled,
		ReadeckURL:                       i.ReadeckURL,
		ReadeckAPIKey:                    i.ReadeckAPIKey,
		ReadeckLabels:                    i.ReadeckLabels,
		ReadeckOnlyURL:                   i.ReadeckOnlyURL,
		ShioriEnabled:                    i.ShioriEnabled,
		ShioriURL:                        i.ShioriURL,
		ShioriUsername:                   i.ShioriUsername,
		ShioriPassword:                   i.ShioriPassword,
		ShaarliEnabled:                   i.ShaarliEnabled,
		ShaarliURL:                       i.ShaarliURL,
		ShaarliAPISecret:                 i.ShaarliAPISecret,
		WebhookEnabled:                   i.WebhookEnabled,
		WebhookURL:                       i.WebhookURL,
		WebhookSecret:                    i.WebhookSecret,
		RSSBridgeEnabled:                 i.RSSBridgeEnabled,
		RSSBridgeURL:                     i.RSSBridgeURL,
		RSSBridgeToken:                   i.RSSBridgeToken,
		OmnivoreEnabled:                  i.OmnivoreEnabled,
		OmnivoreAPIKey:                   i.OmnivoreAPIKey,
		OmnivoreURL:                      i.OmnivoreURL,
		KarakeepEnabled:                  i.KarakeepEnabled,
		KarakeepAPIKey:                   i.KarakeepAPIKey,
		KarakeepURL:                      i.KarakeepURL,
		RaindropEnabled:                  i.RaindropEnabled,
		RaindropToken:                    i.RaindropToken,
		RaindropCollectionID:             i.RaindropCollectionID,
		RaindropTags:                     i.RaindropTags,
		BetulaEnabled:                    i.BetulaEnabled,
		BetulaURL:                        i.BetulaURL,
		BetulaToken:                      i.BetulaToken,
		NtfyEnabled:                      i.NtfyEnabled,
		NtfyTopic:                        i.NtfyTopic,
		NtfyURL:                          i.NtfyURL,
		NtfyAPIToken:                     i.NtfyAPIToken,
		NtfyUsername:                     i.NtfyUsername,
		NtfyPassword:                     i.NtfyPassword,
		NtfyIconURL:                      i.NtfyIconURL,
		NtfyInternalLinks:                i.NtfyInternalLinks,
		CuboxEnabled:                     i.CuboxEnabled,
		CuboxAPILink:                     i.CuboxAPILink,
		DiscordEnabled:                   i.DiscordEnabled,
		DiscordWebhookLink:               i.DiscordWebhookLink,
		SlackEnabled:                     i.SlackEnabled,
		SlackWebhookLink:                 i.SlackWebhookLink,
		PushoverEnabled:                  i.PushoverEnabled,
		PushoverUser:                     i.PushoverUser,
		PushoverToken:                    i.PushoverToken,
		PushoverDevice:                   i.PushoverDevice,
		PushoverPrefix:                   i.PushoverPrefix,
	}

	v.Set("menu", "settings").
		Set("form", f)
	html.OK(w, r, v.Render("integrations"))
}
