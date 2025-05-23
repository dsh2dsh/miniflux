// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

// Integration represents user integration settings.
type Integration struct {
	UserID                           int64            `db:"user_id"`
	BetulaEnabled                    bool             `db:"betula_enabled"`
	BetulaURL                        string           `db:"betula_url"`
	BetulaToken                      string           `db:"betula_token"`
	PinboardEnabled                  bool             `db:"pinboard_enabled"`
	PinboardToken                    string           `db:"pinboard_token"`
	PinboardTags                     string           `db:"pinboard_tags"`
	PinboardMarkAsUnread             bool             `db:"pinboard_mark_as_unread"`
	InstapaperEnabled                bool             `db:"instapaper_enabled"`
	InstapaperUsername               string           `db:"instapaper_username"`
	InstapaperPassword               string           `db:"instapaper_password"`
	FeverEnabled                     bool             `db:"fever_enabled"`
	FeverUsername                    string           `db:"fever_username"`
	FeverToken                       string           `db:"fever_token"`
	GoogleReaderEnabled              bool             `db:"googlereader_enabled"`
	GoogleReaderUsername             string           `db:"googlereader_username"`
	GoogleReaderPassword             string           `db:"googlereader_password"`
	WallabagEnabled                  bool             `db:"wallabag_enabled"`
	WallabagOnlyURL                  bool             `db:"wallabag_only_url"`
	WallabagURL                      string           `db:"wallabag_url"`
	WallabagClientID                 string           `db:"wallabag_client_id"`
	WallabagClientSecret             string           `db:"wallabag_client_secret"`
	WallabagUsername                 string           `db:"wallabag_username"`
	WallabagPassword                 string           `db:"wallabag_password"`
	NunuxKeeperEnabled               bool             `db:"nunux_keeper_enabled"`
	NunuxKeeperURL                   string           `db:"nunux_keeper_url"`
	NunuxKeeperAPIKey                string           `db:"nunux_keeper_api_key"`
	NotionEnabled                    bool             `db:"notion_enabled"`
	NotionToken                      string           `db:"notion_token"`
	NotionPageID                     string           `db:"notion_page_id"`
	EspialEnabled                    bool             `db:"espial_enabled"`
	EspialURL                        string           `db:"espial_url"`
	EspialAPIKey                     string           `db:"espial_api_key"`
	EspialTags                       string           `db:"espial_tags"`
	ReadwiseEnabled                  bool             `db:"readwise_enabled"`
	ReadwiseAPIKey                   string           `db:"readwise_api_key"`
	PocketEnabled                    bool             `db:"pocket_enabled"`
	PocketAccessToken                string           `db:"pocket_access_token"`
	PocketConsumerKey                string           `db:"pocket_consumer_key"`
	TelegramBotEnabled               bool             `db:"telegram_bot_enabled"`
	TelegramBotToken                 string           `db:"telegram_bot_token"`
	TelegramBotChatID                string           `db:"telegram_bot_chat_id"`
	TelegramBotTopicID               *int64           `db:"telegram_bot_topic_id"`
	TelegramBotDisableWebPagePreview bool             `db:"telegram_bot_disable_web_page_preview"`
	TelegramBotDisableNotification   bool             `db:"telegram_bot_disable_notification"`
	TelegramBotDisableButtons        bool             `db:"telegram_bot_disable_buttons"`
	LinkAceEnabled                   bool             `db:"linkace_enabled"`
	LinkAceURL                       string           `db:"linkace_url"`
	LinkAceAPIKey                    string           `db:"linkace_api_key"`
	LinkAceTags                      string           `db:"linkace_tags"`
	LinkAcePrivate                   bool             `db:"linkace_is_private"`
	LinkAceCheckDisabled             bool             `db:"linkace_check_disabled"`
	LinkdingEnabled                  bool             `db:"linkding_enabled"`
	LinkdingURL                      string           `db:"linkding_url"`
	LinkdingAPIKey                   string           `db:"linkding_api_key"`
	LinkdingTags                     string           `db:"linkding_tags"`
	LinkdingMarkAsUnread             bool             `db:"linkding_mark_as_unread"`
	LinkwardenEnabled                bool             `db:"linkwarden_enabled"`
	LinkwardenURL                    string           `db:"linkwarden_url"`
	LinkwardenAPIKey                 string           `db:"linkwarden_api_key"`
	MatrixBotEnabled                 bool             `db:"matrix_bot_enabled"`
	MatrixBotUser                    string           `db:"matrix_bot_user"`
	MatrixBotPassword                string           `db:"matrix_bot_password"`
	MatrixBotURL                     string           `db:"matrix_bot_url"`
	MatrixBotChatID                  string           `db:"matrix_bot_chat_id"`
	AppriseEnabled                   bool             `db:"apprise_enabled"`
	AppriseURL                       string           `db:"apprise_url"`
	AppriseServicesURL               string           `db:"apprise_services_url"`
	ReadeckEnabled                   bool             `db:"readeck_enabled"`
	ReadeckURL                       string           `db:"readeck_url"`
	ReadeckAPIKey                    string           `db:"readeck_api_key"`
	ReadeckLabels                    string           `db:"readeck_labels"`
	ReadeckOnlyURL                   bool             `db:"readeck_only_url"`
	ShioriEnabled                    bool             `db:"shiori_enabled"`
	ShioriURL                        string           `db:"shiori_url"`
	ShioriUsername                   string           `db:"shiori_username"`
	ShioriPassword                   string           `db:"shiori_password"`
	ShaarliEnabled                   bool             `db:"shaarli_enabled"`
	ShaarliURL                       string           `db:"shaarli_url"`
	ShaarliAPISecret                 string           `db:"shaarli_api_secret"`
	WebhookEnabled                   bool             `db:"webhook_enabled"`
	WebhookURL                       string           `db:"webhook_url"`
	WebhookSecret                    string           `db:"webhook_secret"`
	RSSBridgeEnabled                 bool             `db:"rssbridge_enabled"`
	RSSBridgeURL                     string           `db:"rssbridge_url"`
	OmnivoreEnabled                  bool             `db:"omnivore_enabled"`
	OmnivoreAPIKey                   string           `db:"omnivore_api_key"`
	OmnivoreURL                      string           `db:"omnivore_url"`
	RaindropEnabled                  bool             `db:"raindrop_enabled"`
	RaindropToken                    string           `db:"raindrop_token"`
	RaindropCollectionID             string           `db:"raindrop_collection_id"`
	RaindropTags                     string           `db:"raindrop_tags"`
	NtfyEnabled                      bool             `db:"ntfy_enabled"`
	NtfyTopic                        string           `db:"ntfy_topic"`
	NtfyURL                          string           `db:"ntfy_url"`
	NtfyAPIToken                     string           `db:"ntfy_api_token"`
	NtfyUsername                     string           `db:"ntfy_username"`
	NtfyPassword                     string           `db:"ntfy_password"`
	NtfyIconURL                      string           `db:"ntfy_icon_url"`
	NtfyInternalLinks                bool             `db:"ntfy_internal_links"`
	CuboxEnabled                     bool             `db:"cubox_enabled"`
	CuboxAPILink                     string           `db:"cubox_api_link"`
	DiscordEnabled                   bool             `db:"discord_enabled"`
	DiscordWebhookLink               string           `db:"discord_webhook_link"`
	SlackEnabled                     bool             `db:"slack_enabled"`
	SlackWebhookLink                 string           `db:"slack_webhook_link"`
	PushoverEnabled                  bool             `db:"pushover_enabled"`
	PushoverUser                     string           `db:"pushover_user"`
	PushoverToken                    string           `db:"pushover_token"`
	PushoverDevice                   string           `db:"pushover_device"`
	PushoverPrefix                   string           `db:"pushover_prefix"`
	Extra                            IntegrationExtra `db:"extra"`
}

type IntegrationExtra struct {
	RSSBridgeToken string `json:"rssbridge_token,omitempty"`
}
