// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

// Integration represents user integration settings.
type Integration struct {
	AppriseEnabled                   bool   `json:"apprise_enabled,omitempty"`
	AppriseServicesURL               string `json:"apprise_services_url,omitempty"`
	AppriseURL                       string `json:"apprise_url,omitempty"`
	BetulaEnabled                    bool   `json:"betula_enabled,omitempty"`
	BetulaToken                      string `json:"betula_token,omitempty"`
	BetulaURL                        string `json:"betula_url,omitempty"`
	CuboxAPILink                     string `json:"cubox_api_link,omitempty"`
	CuboxEnabled                     bool   `json:"cubox_enabled,omitempty"`
	DiscordEnabled                   bool   `json:"discord_enabled,omitempty"`
	DiscordWebhookLink               string `json:"discord_webhook_link,omitempty"`
	EspialAPIKey                     string `json:"espial_api_key,omitempty"`
	EspialEnabled                    bool   `json:"espial_enabled,omitempty"`
	EspialTags                       string `json:"espial_tags,omitempty"`
	EspialURL                        string `json:"espial_url,omitempty"`
	FeverEnabled                     bool   `json:"fever_enabled,omitempty"`
	FeverToken                       string `json:"fever_token,omitempty"`
	GoogleReaderEnabled              bool   `json:"googlereader_enabled,omitempty"`
	GoogleReaderPassword             string `json:"googlereader_password,omitempty"`
	InstapaperEnabled                bool   `json:"instapaper_enabled,omitempty"`
	InstapaperPassword               string `json:"instapaper_password,omitempty"`
	InstapaperUsername               string `json:"instapaper_username,omitempty"`
	KarakeepAPIKey                   string `json:"karakeep_api_key,omitempty"`
	KarakeepEnabled                  bool   `json:"karakeep_enabled,omitempty"`
	KarakeepURL                      string `json:"karakeep_url,omitempty"`
	LinkAceAPIKey                    string `json:"linkace_api_key,omitempty"`
	LinkAceCheckDisabled             bool   `json:"linkace_check_disabled,omitempty"`
	LinkAceEnabled                   bool   `json:"linkace_enabled,omitempty"`
	LinkAcePrivate                   bool   `json:"linkace_is_private,omitempty"`
	LinkAceTags                      string `json:"linkace_tags,omitempty"`
	LinkAceURL                       string `json:"linkace_url,omitempty"`
	LinkdingAPIKey                   string `json:"linkding_api_key,omitempty"`
	LinkdingEnabled                  bool   `json:"linkding_enabled,omitempty"`
	LinkdingMarkAsUnread             bool   `json:"linkding_mark_as_unread,omitempty"`
	LinkdingTags                     string `json:"linkding_tags,omitempty"`
	LinkdingURL                      string `json:"linkding_url,omitempty"`
	LinkwardenAPIKey                 string `json:"linkwarden_api_key,omitempty"`
	LinkwardenEnabled                bool   `json:"linkwarden_enabled,omitempty"`
	LinkwardenURL                    string `json:"linkwarden_url,omitempty"`
	MatrixBotChatID                  string `json:"matrix_bot_chat_id,omitempty"`
	MatrixBotEnabled                 bool   `json:"matrix_bot_enabled,omitempty"`
	MatrixBotPassword                string `json:"matrix_bot_password,omitempty"`
	MatrixBotURL                     string `json:"matrix_bot_url,omitempty"`
	MatrixBotUser                    string `json:"matrix_bot_user,omitempty"`
	NotionEnabled                    bool   `json:"notion_enabled,omitempty"`
	NotionPageID                     string `json:"notion_page_id,omitempty"`
	NotionToken                      string `json:"notion_token,omitempty"`
	NtfyAPIToken                     string `json:"ntfy_api_token,omitempty"`
	NtfyEnabled                      bool   `json:"ntfy_enabled,omitempty"`
	NtfyIconURL                      string `json:"ntfy_icon_url,omitempty"`
	NtfyInternalLinks                bool   `json:"ntfy_internal_links,omitempty"`
	NtfyPassword                     string `json:"ntfy_password,omitempty"`
	NtfyTopic                        string `json:"ntfy_topic,omitempty"`
	NtfyURL                          string `json:"ntfy_url,omitempty"`
	NtfyUsername                     string `json:"ntfy_username,omitempty"`
	NunuxKeeperAPIKey                string `json:"nunux_keeper_api_key,omitempty"`
	NunuxKeeperEnabled               bool   `json:"nunux_keeper_enabled,omitempty"`
	NunuxKeeperURL                   string `json:"nunux_keeper_url,omitempty"`
	OmnivoreAPIKey                   string `json:"omnivore_api_key,omitempty"`
	OmnivoreEnabled                  bool   `json:"omnivore_enabled,omitempty"`
	OmnivoreURL                      string `json:"omnivore_url,omitempty"`
	PinboardEnabled                  bool   `json:"pinboard_enabled,omitempty"`
	PinboardMarkAsUnread             bool   `json:"pinboard_mark_as_unread,omitempty"`
	PinboardTags                     string `json:"pinboard_tags,omitempty"`
	PinboardToken                    string `json:"pinboard_token,omitempty"`
	PushoverDevice                   string `json:"pushover_device,omitempty"`
	PushoverEnabled                  bool   `json:"pushover_enabled,omitempty"`
	PushoverPrefix                   string `json:"pushover_prefix,omitempty"`
	PushoverToken                    string `json:"pushover_token,omitempty"`
	PushoverUser                     string `json:"pushover_user,omitempty"`
	RSSBridgeEnabled                 bool   `json:"rssbridge_enabled,omitempty"`
	RSSBridgeToken                   string `json:"rssbridge_token,omitempty"`
	RSSBridgeURL                     string `json:"rssbridge_url,omitempty"`
	RaindropCollectionID             string `json:"raindrop_collection_id,omitempty"`
	RaindropEnabled                  bool   `json:"raindrop_enabled,omitempty"`
	RaindropTags                     string `json:"raindrop_tags,omitempty"`
	RaindropToken                    string `json:"raindrop_token,omitempty"`
	ReadeckAPIKey                    string `json:"readeck_api_key,omitempty"`
	ReadeckEnabled                   bool   `json:"readeck_enabled,omitempty"`
	ReadeckLabels                    string `json:"readeck_labels,omitempty"`
	ReadeckOnlyURL                   bool   `json:"readeck_only_url,omitempty"`
	ReadeckURL                       string `json:"readeck_url,omitempty"`
	ReadwiseAPIKey                   string `json:"readwise_api_key,omitempty"`
	ReadwiseEnabled                  bool   `json:"readwise_enabled,omitempty"`
	ShaarliAPISecret                 string `json:"shaarli_api_secret,omitempty"`
	ShaarliEnabled                   bool   `json:"shaarli_enabled,omitempty"`
	ShaarliURL                       string `json:"shaarli_url,omitempty"`
	ShioriEnabled                    bool   `json:"shiori_enabled,omitempty"`
	ShioriPassword                   string `json:"shiori_password,omitempty"`
	ShioriURL                        string `json:"shiori_url,omitempty"`
	ShioriUsername                   string `json:"shiori_username,omitempty"`
	SlackEnabled                     bool   `json:"slack_enabled,omitempty"`
	SlackWebhookLink                 string `json:"slack_webhook_link,omitempty"`
	TelegramBotChatID                string `json:"telegram_bot_chat_id,omitempty"`
	TelegramBotDisableButtons        bool   `json:"telegram_bot_disable_buttons,omitempty"`
	TelegramBotDisableNotification   bool   `json:"telegram_bot_disable_notification,omitempty"`
	TelegramBotDisableWebPagePreview bool   `json:"telegram_bot_disable_web_page_preview,omitempty"`
	TelegramBotEnabled               bool   `json:"telegram_bot_enabled,omitempty"`
	TelegramBotToken                 string `json:"telegram_bot_token,omitempty"`
	TelegramBotTopicID               *int64 `json:"telegram_bot_topic_id,omitempty"`
	WallabagClientID                 string `json:"wallabag_client_id,omitempty"`
	WallabagClientSecret             string `json:"wallabag_client_secret,omitempty"`
	WallabagEnabled                  bool   `json:"wallabag_enabled,omitempty"`
	WallabagOnlyURL                  bool   `json:"wallabag_only_url,omitempty"`
	WallabagPassword                 string `json:"wallabag_password,omitempty"`
	WallabagURL                      string `json:"wallabag_url,omitempty"`
	WallabagUsername                 string `json:"wallabag_username,omitempty"`
	WebhookEnabled                   bool   `json:"webhook_enabled,omitempty"`
	WebhookSecret                    string `json:"webhook_secret,omitempty"`
	WebhookURL                       string `json:"webhook_url,omitempty"`
}

func (self *Integration) RSSBridgeURLIfEnabled() string {
	if self.RSSBridgeEnabled {
		return self.RSSBridgeURL
	}
	return ""
}

func (self *Integration) RSSBridgeTokenIfEnabled() string {
	if self.RSSBridgeEnabled {
		return self.RSSBridgeToken
	}
	return ""
}
