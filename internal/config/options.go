// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config // import "miniflux.app/v2/internal/config"

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/url"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"

	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/version"
)

const (
	defaultBaseURL     = "http://localhost"
	defaultDatabaseURL = "user=postgres password=postgres dbname=miniflux2 sslmode=disable"
)

var defaultUA = "Miniflux/dsh2dsh-" + version.Version +
	" (https://github.com/dsh2dsh/miniflux)"

// Option contains a key to value map of a single option. It may be used to
// output debug strings.
type Option struct {
	Key   string
	Value any
}

// Options contains configuration options.
type Options struct {
	HostLimits map[string]HostLimits `yaml:"host_limits" validate:"dive,keys,required,endkeys,required"`

	env EnvOptions

	rootURL              string
	basePath             string
	mediaProxyPrivateKey []byte
	trustedProxies       map[string]struct{}
}

type HostLimits struct {
	Connections int64   `yaml:"connections" validate:"omitempty,min=0"`
	Rate        float64 `yaml:"rate" validate:"omitempty,min=0"`
}

func (self *HostLimits) withDefaults(connections int64, rate float64,
) HostLimits {
	limits := *self
	if limits.Connections == 0 {
		limits.Connections = connections
	}
	if limits.Rate == 0 {
		limits.Rate = rate
	}
	return limits
}

type EnvOptions struct {
	DisableHSTS                    bool     `env:"DISABLE_HSTS"`
	DisableHttpService             bool     `env:"DISABLE_HTTP_SERVICE"`
	DisableScheduler               bool     `env:"DISABLE_SCHEDULER_SERVICE"`
	HTTPS                          bool     `env:"HTTPS"`
	LogFile                        string   `env:"LOG_FILE" validate:"required"`
	LogDateTime                    bool     `env:"LOG_DATE_TIME"`
	LogFormat                      string   `env:"LOG_FORMAT" validate:"required,oneof=human json text"`
	LogLevel                       string   `env:"LOG_LEVEL" validate:"required,oneof=debug info warning error"`
	Logging                        []Log    `envPrefix:"LOG" validate:"dive,required"`
	BaseURL                        string   `env:"BASE_URL" validate:"required"`
	DatabaseURL                    string   `env:"DATABASE_URL" validate:"required"`
	DatabaseURLFile                *string  `env:"DATABASE_URL_FILE,file"`
	DatabaseMaxConns               int      `env:"DATABASE_MAX_CONNS" validate:"min=1"`
	DatabaseMinConns               int      `env:"DATABASE_MIN_CONNS" validate:"min=0"`
	DatabaseConnectionLifetime     int      `env:"DATABASE_CONNECTION_LIFETIME" validate:"gt=0"`
	RunMigrations                  bool     `env:"RUN_MIGRATIONS"`
	ListenAddr                     string   `env:"LISTEN_ADDR" validate:"required,hostname|hostname_port"`
	Port                           string   `env:"PORT"`
	CertFile                       string   `env:"CERT_FILE" validate:"omitempty,filepath"`
	CertDomain                     string   `env:"CERT_DOMAIN"`
	CertKeyFile                    string   `env:"KEY_FILE" validate:"omitempty,filepath"`
	CleanupFrequencyHours          int      `env:"CLEANUP_FREQUENCY_HOURS" validate:"min=1"`
	CleanupArchiveReadDays         int      `env:"CLEANUP_ARCHIVE_READ_DAYS" validate:"min=0"`
	CleanupArchiveUnreadDays       int      `env:"CLEANUP_ARCHIVE_UNREAD_DAYS" validate:"min=0"`
	CleanupArchiveBatchSize        int      `env:"CLEANUP_ARCHIVE_BATCH_SIZE" validate:"min=1"`
	CleanupRemoveSessionsDays      int      `env:"CLEANUP_REMOVE_SESSIONS_DAYS" validate:"min=0"`
	CleanupInactiveSessionsDays    int      `env:"CLEANUP_INACTIVE_SESSIONS_DAYS" validate:"min=0"`
	PollingFrequency               int      `env:"POLLING_FREQUENCY" validate:"min=1"`
	ForceRefreshInterval           int      `env:"FORCE_REFRESH_INTERVAL" validate:"min=0"`
	BatchSize                      int      `env:"BATCH_SIZE" validate:"min=1"`
	SchedulerRoundRobinMinInterval int      `env:"SCHEDULER_ROUND_ROBIN_MIN_INTERVAL" validate:"min=1"`
	SchedulerRoundRobinMaxInterval int      `env:"SCHEDULER_ROUND_ROBIN_MAX_INTERVAL" validate:"min=1"`
	PollingParsingErrorLimit       int      `env:"POLLING_PARSING_ERROR_LIMIT" validate:"min=0"`
	WorkerPoolSize                 int      `env:"WORKER_POOL_SIZE" validate:"min=1"`
	CreateAdmin                    bool     `env:"CREATE_ADMIN"`
	AdminUsername                  string   `env:"ADMIN_USERNAME"`
	AdminUsernameFile              *string  `env:"ADMIN_USERNAME_FILE,file"`
	AdminPassword                  string   `env:"ADMIN_PASSWORD"`
	AdminPasswordFile              *string  `env:"ADMIN_PASSWORD_FILE,file"`
	MediaProxyHTTPClientTimeout    int      `env:"MEDIA_PROXY_HTTP_CLIENT_TIMEOUT" validate:"min=1"`
	MediaProxyMode                 string   `env:"MEDIA_PROXY_MODE" validate:"required,oneof=none http-only all"`
	MediaProxyResourceTypes        []string `env:"MEDIA_PROXY_RESOURCE_TYPES" validate:"omitempty,dive,oneof=image video audio"`
	MediaProxyCustomURL            *url.URL `env:"MEDIA_PROXY_CUSTOM_URL"`
	FetchBilibiliWatchTime         bool     `env:"FETCH_BILIBILI_WATCH_TIME"`
	FetchNebulaWatchTime           bool     `env:"FETCH_NEBULA_WATCH_TIME"`
	FetchOdyseeWatchTime           bool     `env:"FETCH_ODYSEE_WATCH_TIME"`
	FetchYouTubeWatchTime          bool     `env:"FETCH_YOUTUBE_WATCH_TIME"`
	FilterEntryMaxAgeDays          int      `env:"FILTER_ENTRY_MAX_AGE_DAYS" validate:"min=0"`
	YouTubeApiKey                  string   `env:"YOUTUBE_API_KEY"`
	YouTubeEmbedUrlOverride        *url.URL `env:"YOUTUBE_EMBED_URL_OVERRIDE" envDefault:"https://www.youtube-nocookie.com/embed/"`
	Oauth2UserCreationAllowed      bool     `env:"OAUTH2_USER_CREATION"`
	Oauth2ClientID                 string   `env:"OAUTH2_CLIENT_ID"`
	Oauth2ClientIDFile             *string  `env:"OAUTH2_CLIENT_ID_FILE,file"`
	Oauth2ClientSecret             string   `env:"OAUTH2_CLIENT_SECRET"`
	Oauth2ClientSecretFile         *string  `env:"OAUTH2_CLIENT_SECRET_FILE,file"`
	Oauth2RedirectURL              string   `env:"OAUTH2_REDIRECT_URL" validate:"omitempty,url"`
	OidcDiscoveryEndpoint          string   `env:"OAUTH2_OIDC_DISCOVERY_ENDPOINT" validate:"omitempty,url"`
	OidcProviderName               string   `env:"OAUTH2_OIDC_PROVIDER_NAME"`
	Oauth2Provider                 string   `env:"OAUTH2_PROVIDER" validate:"omitempty,oneof=oidc google"`
	DisableLocalAuth               bool     `env:"DISABLE_LOCAL_AUTH"`
	HttpClientTimeout              int      `env:"HTTP_CLIENT_TIMEOUT" validate:"min=1"`
	HttpClientMaxBodySize          int64    `env:"HTTP_CLIENT_MAX_BODY_SIZE" validate:"min=1"`
	HttpClientProxyURL             *url.URL `env:"HTTP_CLIENT_PROXY"`
	HttpClientProxies              []string `env:"HTTP_CLIENT_PROXIES" validate:"dive,required,url"`
	HttpClientUserAgent            string   `env:"HTTP_CLIENT_USER_AGENT"`
	HttpServerTimeout              int      `env:"HTTP_SERVER_TIMEOUT" validate:"min=1"`
	AuthProxyHeader                string   `env:"AUTH_PROXY_HEADER"`
	AuthProxyUserCreation          bool     `env:"AUTH_PROXY_USER_CREATION"`
	MaintenanceMode                bool     `env:"MAINTENANCE_MODE"`
	MaintenanceMessage             string   `env:"MAINTENANCE_MESSAGE" validate:"required_with=MaintenanceMode"`
	MetricsCollector               bool     `env:"METRICS_COLLECTOR"`
	MetricsRefreshInterval         int      `env:"METRICS_REFRESH_INTERVAL" validate:"min=1"`
	MetricsAllowedNetworks         []string `env:"METRICS_ALLOWED_NETWORKS" validate:"dive,required"`
	MetricsUsername                string   `env:"METRICS_USERNAME"`
	MetricsUsernameFile            *string  `env:"METRICS_USERNAME_FILE,file"`
	MetricsPassword                string   `env:"METRICS_PASSWORD"`
	MetricsPasswordFile            *string  `env:"METRICS_PASSWORD_FILE,file"`
	Watchdog                       bool     `env:"WATCHDOG"`
	InvidiousInstance              string   `env:"INVIDIOUS_INSTANCE"`
	MediaProxyPrivateKey           string   `env:"MEDIA_PROXY_PRIVATE_KEY"`
	WebAuthn                       bool     `env:"WEBAUTHN"`
	PreferSiteIcon                 bool     `env:"PREFER_SITE_ICON"`
	ConnectionsPerSever            int64    `env:"CONNECTIONS_PER_SERVER" validate:"min=0"`
	RateLimitPerServer             float64  `env:"RATE_LIMIT_PER_SERVER" validate:"min=0"`
	TrustedProxies                 []string `env:"TRUSTED_PROXIES" validate:"dive,required,ip"`
	Testing                        bool     `env:"TESTING"`
	Operators                      []string `env:"OPERATORS"`
}

type Log struct {
	LogFile     string `env:"FILE" validate:"required"`
	LogDateTime bool   `env:"DATE_TIME"`
	LogFormat   string `env:"FORMAT" validate:"required,oneof=human json text"`
	LogLevel    string `env:"LEVEL" validate:"required,oneof=debug info warning error"`
}

// NewOptions returns Options with default values.
func NewOptions() *Options {
	maxConns := max(4, runtime.GOMAXPROCS(0))

	return &Options{
		HostLimits: map[string]HostLimits{},

		env: EnvOptions{
			LogFile:                        "stderr",
			LogFormat:                      "text",
			LogLevel:                       "info",
			BaseURL:                        defaultBaseURL,
			DatabaseURL:                    defaultDatabaseURL,
			DatabaseMaxConns:               maxConns,
			DatabaseMinConns:               0,
			DatabaseConnectionLifetime:     60,
			ListenAddr:                     "127.0.0.1:8080",
			CleanupFrequencyHours:          24,
			CleanupArchiveReadDays:         60,
			CleanupArchiveUnreadDays:       180,
			CleanupArchiveBatchSize:        10000,
			CleanupRemoveSessionsDays:      30,
			CleanupInactiveSessionsDays:    10,
			PollingFrequency:               60,
			ForceRefreshInterval:           30,
			BatchSize:                      100,
			SchedulerRoundRobinMinInterval: 60,
			SchedulerRoundRobinMaxInterval: 1440,
			PollingParsingErrorLimit:       3,
			WorkerPoolSize:                 16,
			MediaProxyHTTPClientTimeout:    120,
			MediaProxyMode:                 "http-only",
			MediaProxyResourceTypes:        []string{"image"},
			OidcProviderName:               "OpenID Connect",
			HttpClientTimeout:              20,
			HttpClientMaxBodySize:          15,
			HttpClientProxies:              []string{},
			HttpClientUserAgent:            defaultUA,
			HttpServerTimeout:              300,
			MaintenanceMessage:             "Miniflux is currently under maintenance",
			MetricsRefreshInterval:         60,
			MetricsAllowedNetworks:         []string{"127.0.0.1/8"},
			Watchdog:                       true,
			InvidiousInstance:              "yewtu.be",
			ConnectionsPerSever:            8,
			RateLimitPerServer:             10,
			TrustedProxies:                 []string{"127.0.0.1"},
		},

		rootURL: defaultBaseURL,
	}
}

func (o *Options) init() (err error) {
	if o.env.Port != "" {
		o.env.ListenAddr = ":" + o.env.Port
	}

	if err := o.validate(); err != nil {
		return err
	}

	o.env.HttpClientMaxBodySize *= 1024 * 1024
	o.env.MediaProxyResourceTypes = uniqStringList(o.env.MediaProxyResourceTypes)

	o.applyFileStrings()
	if err = o.applyPrivateKeys(); err != nil {
		return err
	}
	o.makeTrustedProxies()

	o.env.BaseURL, o.rootURL, o.basePath, err = parseBaseURL(o.env.BaseURL)
	return err
}

func (o *Options) validate() error {
	if err := Validator().Struct(&o.env); err != nil {
		return fmt.Errorf("config: failed validate: %w", err)
	}

	if o.DisableLocalAuth() {
		switch {
		case o.OAuth2Provider() == "" && o.AuthProxyHeader() == "":
			return errors.New("DISABLE_LOCAL_AUTH is enabled but neither OAUTH2_PROVIDER nor AUTH_PROXY_HEADER is not set. Please enable at least one authentication source")
		case o.OAuth2Provider() != "" && !o.IsOAuth2UserCreationAllowed():
			return errors.New("DISABLE_LOCAL_AUTH is enabled and an OAUTH2_PROVIDER is configured, but OAUTH2_USER_CREATION is not enabled")
		case o.AuthProxyHeader() != "" && !o.IsAuthProxyUserCreationAllowed():
			return errors.New("DISABLE_LOCAL_AUTH is enabled and an AUTH_PROXY_HEADER is configured, but AUTH_PROXY_USER_CREATION is not enabled")
		}
	}
	return nil
}

func uniqStringList(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	for i, s := range items {
		s = strings.TrimSpace(s)
		if s != "" {
			if _, found := seen[s]; !found {
				seen[s] = struct{}{}
			} else {
				s = ""
			}
		}
		items[i] = s
	}
	if len(seen) < len(items) {
		items = slices.DeleteFunc(items, func(s string) bool { return s == "" })
	}
	return items
}

func (o *Options) applyFileStrings() {
	opts := []struct {
		From *string
		To   *string
	}{
		{o.env.DatabaseURLFile, &o.env.DatabaseURL},
		{o.env.AdminPasswordFile, &o.env.AdminPassword},
		{o.env.AdminUsernameFile, &o.env.AdminUsername},
		{o.env.MetricsPasswordFile, &o.env.MetricsPassword},
		{o.env.MetricsUsernameFile, &o.env.MetricsUsername},
		{o.env.Oauth2ClientIDFile, &o.env.Oauth2ClientID},
		{o.env.Oauth2ClientSecretFile, &o.env.Oauth2ClientSecret},
	}
	for _, opt := range opts {
		if opt.From != nil {
			*opt.To = *opt.From
		}
	}
}

func (o *Options) applyPrivateKeys() error {
	opts := []struct {
		From       string
		To         *[]byte
		Deprecated string
	}{
		{
			From: o.env.MediaProxyPrivateKey,
			To:   &o.mediaProxyPrivateKey,
		},
	}

	for _, opt := range opts {
		switch {
		case opt.From != "":
			if opt.Deprecated != "" {
				slog.Warn(opt.Deprecated)
			}
			*opt.To = []byte(opt.From)
		case opt.Deprecated == "":
			*opt.To = crypto.GenerateRandomBytes(16)
		}
	}
	return nil
}

func (o *Options) makeTrustedProxies() {
	n := len(o.env.TrustedProxies)
	if n == 0 {
		o.trustedProxies = make(map[string]struct{})
		return
	}

	o.trustedProxies = make(map[string]struct{}, n)
	for _, ip := range o.env.TrustedProxies {
		o.trustedProxies[ip] = struct{}{}
	}
}

func Validator() *validator.Validate {
	if validate == nil {
		validate = validator.New(validator.WithRequiredStructEnabled())
		validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
			if s := fld.Tag.Get("env"); s != "" {
				name, _, _ := strings.Cut(fld.Tag.Get("env"), ",")
				if name == "-" {
					return ""
				}
				return name
			}
			name, _, _ := strings.Cut(fld.Tag.Get("yaml"), ",")
			if name == "-" {
				return ""
			}
			return name
		})
	}
	return validate
}

var validate *validator.Validate

func (o *Options) HTTPS() bool  { return o.env.HTTPS }
func (o *Options) EnableHTTPS() { o.env.HTTPS = true }

func (o *Options) LogFile() string { return o.env.LogFile }

// LogDateTime returns true if the date/time should be displayed in log
// messages.
func (o *Options) LogDateTime() bool { return o.env.LogDateTime }

// LogFormat returns the log format.
func (o *Options) LogFormat() string { return o.env.LogFormat }

// LogLevel returns the log level.
func (o *Options) LogLevel() string { return o.env.LogLevel }

// SetLogLevel sets the log level.
func (o *Options) SetLogLevel(level string) { o.env.LogLevel = level }

// HasMaintenanceMode returns true if maintenance mode is enabled.
func (o *Options) HasMaintenanceMode() bool { return o.env.MaintenanceMode }

// MaintenanceMessage returns maintenance message.
func (o *Options) MaintenanceMessage() string {
	return o.env.MaintenanceMessage
}

// BaseURL returns the application base URL with path.
func (o *Options) BaseURL() string { return o.env.BaseURL }

// RootURL returns the base URL without path.
func (o *Options) RootURL() string { return o.rootURL }

// BasePath returns the application base path according to the base URL.
func (o *Options) BasePath() string { return o.basePath }

// IsDefaultDatabaseURL returns true if the default database URL is used.
func (o *Options) IsDefaultDatabaseURL() bool {
	return o.env.DatabaseURL == defaultDatabaseURL
}

// DatabaseURL returns the database URL.
func (o *Options) DatabaseURL() string { return o.env.DatabaseURL }

// DatabaseMaxConns returns the maximum number of database connections.
func (o *Options) DatabaseMaxConns() int { return o.env.DatabaseMaxConns }

// DatabaseMinConns returns the minimum number of database connections.
func (o *Options) DatabaseMinConns() int { return o.env.DatabaseMinConns }

// DatabaseConnectionLifetime returns the maximum amount of time a connection
// may be reused.
func (o *Options) DatabaseConnectionLifetime() time.Duration {
	return time.Duration(o.env.DatabaseConnectionLifetime) * time.Minute
}

// ListenAddr returns the listen address for the HTTP server.
func (o *Options) ListenAddr() string { return o.env.ListenAddr }

// CertFile returns the SSL certificate filename if any.
func (o *Options) CertFile() string { return o.env.CertFile }

// CertKeyFile returns the private key filename for custom SSL certificate.
func (o *Options) CertKeyFile() string { return o.env.CertKeyFile }

// CertDomain returns the domain to use for Let's Encrypt certificate.
func (o *Options) CertDomain() string { return o.env.CertDomain }

// CleanupFrequencyHours returns the interval in hours for cleanup jobs.
func (o *Options) CleanupFrequencyHours() time.Duration {
	return time.Duration(o.env.CleanupFrequencyHours) * time.Hour
}

// CleanupArchiveReadDays returns the number of days after which marking read
// items as removed.
func (o *Options) CleanupArchiveReadDays() int {
	return o.env.CleanupArchiveReadDays
}

// CleanupArchiveUnreadDays returns the number of days after which marking
// unread items as removed.
func (o *Options) CleanupArchiveUnreadDays() int {
	return o.env.CleanupArchiveUnreadDays
}

// CleanupArchiveBatchSize returns the number of entries to archive for each
// interval.
func (o *Options) CleanupArchiveBatchSize() int {
	return o.env.CleanupArchiveBatchSize
}

// CleanupRemoveSessionsDays returns the number of days after which to remove
// sessions.
func (o *Options) CleanupRemoveSessionsDays() int {
	return o.env.CleanupRemoveSessionsDays
}

func (o *Options) CleanupRemoveSessionsInterval() time.Duration {
	return time.Duration(o.CleanupRemoveSessionsDays()) * 24 * time.Hour
}

func (o *Options) CleanupInactiveSessionsDays() int {
	return o.env.CleanupInactiveSessionsDays
}

func (o *Options) CleanupInactiveSessionsInterval() time.Duration {
	return time.Duration(o.env.CleanupInactiveSessionsDays) * 24 * time.Hour
}

// WorkerPoolSize returns the number of background worker.
func (o *Options) WorkerPoolSize() int { return o.env.WorkerPoolSize }

// PollingFrequency returns the interval to refresh feeds in the background.
func (o *Options) PollingFrequency() time.Duration {
	return time.Duration(o.env.PollingFrequency) * time.Minute
}

// ForceRefreshInterval returns the force refresh interval
func (o *Options) ForceRefreshInterval() int {
	return o.env.ForceRefreshInterval
}

// BatchSize returns the number of feeds to send for background processing.
func (o *Options) BatchSize() int { return o.env.BatchSize }

func (o *Options) SchedulerRoundRobinMinInterval() int {
	return o.env.SchedulerRoundRobinMinInterval
}

func (o *Options) SchedulerRoundRobinMaxInterval() int {
	return o.env.SchedulerRoundRobinMaxInterval
}

// PollingParsingErrorLimit returns the limit of errors when to stop polling.
func (o *Options) PollingParsingErrorLimit() int {
	return o.env.PollingParsingErrorLimit
}

// IsOAuth2UserCreationAllowed returns true if user creation is allowed for
// OAuth2 users.
func (o *Options) IsOAuth2UserCreationAllowed() bool {
	return o.env.Oauth2UserCreationAllowed
}

// OAuth2ClientID returns the OAuth2 Client ID.
func (o *Options) OAuth2ClientID() string { return o.env.Oauth2ClientID }

// OAuth2ClientSecret returns the OAuth2 client secret.
func (o *Options) OAuth2ClientSecret() string {
	return o.env.Oauth2ClientSecret
}

// OAuth2RedirectURL returns the OAuth2 redirect URL.
func (o *Options) OAuth2RedirectURL() string { return o.env.Oauth2RedirectURL }

// OIDCDiscoveryEndpoint returns the OAuth2 OIDC discovery endpoint.
func (o *Options) OIDCDiscoveryEndpoint() string {
	return o.env.OidcDiscoveryEndpoint
}

// OIDCProviderName returns the OAuth2 OIDC provider's display name
func (o *Options) OIDCProviderName() string { return o.env.OidcProviderName }

// OAuth2Provider returns the name of the OAuth2 provider configured.
func (o *Options) OAuth2Provider() string { return o.env.Oauth2Provider }

// DisableLocalAUth returns true if the local user database should not be used
// to authenticate users.
func (o *Options) DisableLocalAuth() bool { return o.env.DisableLocalAuth }

// HasHSTS returns true if HTTP Strict Transport Security is enabled.
func (o *Options) HasHSTS() bool { return !o.env.DisableHSTS }

// RunMigrations returns true if the environment variable RUN_MIGRATIONS is not
// empty.
func (o *Options) RunMigrations() bool { return o.env.RunMigrations }

// CreateAdmin returns true if the environment variable CREATE_ADMIN is not
// empty.
func (o *Options) CreateAdmin() bool { return o.env.CreateAdmin }

// AdminUsername returns the admin username if defined.
func (o *Options) AdminUsername() string { return o.env.AdminUsername }

// AdminPassword returns the admin password if defined.
func (o *Options) AdminPassword() string { return o.env.AdminPassword }

// FetchYouTubeWatchTime returns true if the YouTube video duration should be
// fetched and used as a reading time.
func (o *Options) FetchYouTubeWatchTime() bool {
	return o.env.FetchYouTubeWatchTime
}

// YouTubeApiKey returns the YouTube API key if defined.
func (o *Options) YouTubeApiKey() string { return o.env.YouTubeApiKey }

// YouTubeEmbedUrlOverride returns the YouTube embed URL override if defined.
func (o *Options) YouTubeEmbedUrlOverride() string {
	return o.env.YouTubeEmbedUrlOverride.String()
}

// YouTubeEmbedDomain returns the domain used for YouTube embeds.
func (o *Options) YouTubeEmbedDomain() string {
	return o.env.YouTubeEmbedUrlOverride.Hostname()
}

func (o *Options) YouTubeEmbedURL() *url.URL {
	return o.env.YouTubeEmbedUrlOverride
}

// FetchNebulaWatchTime returns true if the Nebula video duration should be
// fetched and used as a reading time.
func (o *Options) FetchNebulaWatchTime() bool { return o.env.FetchNebulaWatchTime }

// FetchOdyseeWatchTime returns true if the Odysee video duration should be
// fetched and used as a reading time.
func (o *Options) FetchOdyseeWatchTime() bool {
	return o.env.FetchOdyseeWatchTime
}

// FetchBilibiliWatchTime returns true if the Bilibili video duration should be
// fetched and used as a reading time.
func (o *Options) FetchBilibiliWatchTime() bool {
	return o.env.FetchBilibiliWatchTime
}

// MediaProxyMode returns "none" to never proxy, "http-only" to proxy non-HTTPS,
// "all" to always proxy.
func (o *Options) MediaProxyMode() string { return o.env.MediaProxyMode }

// MediaProxyResourceTypes returns a slice of resource types to proxy.
func (o *Options) MediaProxyResourceTypes() []string {
	return o.env.MediaProxyResourceTypes
}

// MediaCustomProxyURL returns the custom proxy URL for medias.
func (o *Options) MediaCustomProxyURL() *url.URL {
	return o.env.MediaProxyCustomURL
}

// MediaProxyHTTPClientTimeout returns the time limit in seconds before the
// proxy HTTP client cancel the request.
func (o *Options) MediaProxyHTTPClientTimeout() time.Duration {
	return time.Duration(o.env.MediaProxyHTTPClientTimeout) * time.Minute
}

// MediaProxyPrivateKey returns the private key used by the media proxy.
func (o *Options) MediaProxyPrivateKey() []byte {
	return o.mediaProxyPrivateKey
}

// HasHTTPService returns true if the HTTP service is enabled.
func (o *Options) HasHTTPService() bool { return !o.env.DisableHttpService }

// HasSchedulerService returns true if the scheduler service is enabled.
func (o *Options) HasSchedulerService() bool { return !o.env.DisableScheduler }

// HTTPClientTimeout returns the time limit in seconds before the HTTP client
// cancel the request.
func (o *Options) HTTPClientTimeout() time.Duration {
	return time.Duration(o.env.HttpClientTimeout) * time.Second
}

// HTTPClientMaxBodySize returns the number of bytes allowed for the HTTP client
// to transfer.
func (o *Options) HTTPClientMaxBodySize() int64 {
	return o.env.HttpClientMaxBodySize
}

// HTTPClientProxyURL returns the client HTTP proxy URL if configured.
func (o *Options) HTTPClientProxyURL() *url.URL {
	return o.env.HttpClientProxyURL
}

// HasHTTPClientProxyURLConfigured returns true if the client HTTP proxy URL if
// configured.
func (o *Options) HasHTTPClientProxyURLConfigured() bool {
	return o.env.HttpClientProxyURL != nil
}

// HTTPClientProxies returns the list of proxies.
func (o *Options) HTTPClientProxies() []string {
	return o.env.HttpClientProxies
}

// HTTPClientProxiesString returns true if the list of rotating proxies are
// configured.
func (o *Options) HasHTTPClientProxiesConfigured() bool {
	return len(o.env.HttpClientProxies) > 0
}

// HTTPServerTimeout returns the time limit in seconds before the HTTP server
// cancel the request.
func (o *Options) HTTPServerTimeout() time.Duration {
	return time.Duration(o.env.HttpServerTimeout) * time.Second
}

// AuthProxyHeader returns an HTTP header name that contains username for
// authentication using auth proxy.
func (o *Options) AuthProxyHeader() string { return o.env.AuthProxyHeader }

// IsAuthProxyUserCreationAllowed returns true if user creation is allowed for
// users authenticated using auth proxy.
func (o *Options) IsAuthProxyUserCreationAllowed() bool {
	return o.env.AuthProxyUserCreation
}

// HasMetricsCollector returns true if metrics collection is enabled.
func (o *Options) HasMetricsCollector() bool { return o.env.MetricsCollector }

// MetricsRefreshInterval returns the refresh interval.
func (o *Options) MetricsRefreshInterval() time.Duration {
	return time.Duration(o.env.MetricsRefreshInterval) * time.Second
}

// MetricsAllowedNetworks returns the list of networks allowed to connect to the
// metrics endpoint.
func (o *Options) MetricsAllowedNetworks() []string {
	return o.env.MetricsAllowedNetworks
}

func (o *Options) MetricsUsername() string { return o.env.MetricsUsername }
func (o *Options) MetricsPassword() string { return o.env.MetricsPassword }

// HTTPClientUserAgent returns the global User-Agent header for miniflux.
func (o *Options) HTTPClientUserAgent() string {
	return o.env.HttpClientUserAgent
}

// HasWatchdog returns true if the systemd watchdog is enabled.
func (o *Options) HasWatchdog() bool { return o.env.Watchdog }

// InvidiousInstance returns the invidious instance used by miniflux
func (o *Options) InvidiousInstance() string { return o.env.InvidiousInstance }

// WebAuthn returns true if WebAuthn logins are supported
func (o *Options) WebAuthn() bool { return o.env.WebAuthn }

// FilterEntryMaxAgeDays returns the number of days after which entries should
// be retained.
func (o *Options) FilterEntryMaxAgeDays() int {
	return o.env.FilterEntryMaxAgeDays
}

func (o *Options) PreferSiteIcon() bool { return o.env.PreferSiteIcon }

func (o *Options) ConnectionsPerServer() int64 {
	return o.env.ConnectionsPerSever
}

func (o *Options) RateLimitPerServer() float64 {
	return o.env.RateLimitPerServer
}

func (o *Options) TrustedProxy(ip string) bool {
	_, ok := o.trustedProxies[ip]
	return ok
}

func (o *Options) Operator(username string) bool {
	if len(o.env.Operators) == 0 {
		return false
	}
	return slices.Contains(o.env.Operators, username)
}

func (o *Options) Logging() []Log {
	if len(o.env.Logging) == 0 {
		return []Log{{
			LogFile:     o.LogFile(),
			LogDateTime: o.LogDateTime(),
			LogFormat:   o.LogFormat(),
			LogLevel:    o.LogLevel(),
		}}
	}
	return slices.Clone(o.env.Logging)
}

func (o *Options) FindHostLimits(hostname string) (found HostLimits) {
	for hostname != "" {
		if limits, ok := o.HostLimits[hostname]; ok {
			found = limits
			break
		}
		_, hostname, _ = strings.Cut(hostname, ".")
	}
	return found.withDefaults(o.ConnectionsPerServer(), o.RateLimitPerServer())
}

// SortedOptions returns options as a list of key value pairs, sorted by keys.
func (o *Options) SortedOptions(redactSecret bool) []Option {
	var clientProxyURLRedacted string
	if o.env.HttpClientProxyURL != nil {
		if redactSecret {
			clientProxyURLRedacted = o.env.HttpClientProxyURL.Redacted()
		} else {
			clientProxyURLRedacted = o.env.HttpClientProxyURL.String()
		}
	}

	var clientProxyURLsRedacted string
	if len(o.env.HttpClientProxies) > 0 {
		if redactSecret {
			proxyURLs := make([]string, len(o.env.HttpClientProxies))
			for i := range o.env.HttpClientProxies {
				proxyURLs[i] = "<redacted>"
			}
			clientProxyURLsRedacted = strings.Join(proxyURLs, ",")
		} else {
			clientProxyURLsRedacted = strings.Join(o.env.HttpClientProxies, ",")
		}
	}

	var mediaProxyPrivateKeyValue string
	if len(o.mediaProxyPrivateKey) > 0 {
		mediaProxyPrivateKeyValue = "<binary-data>"
	}

	keyValues := map[string]any{
		"ADMIN_PASSWORD":                     secretValue(o.AdminPassword(), redactSecret),
		"ADMIN_USERNAME":                     o.AdminUsername(),
		"AUTH_PROXY_HEADER":                  o.AuthProxyHeader(),
		"AUTH_PROXY_USER_CREATION":           o.IsAuthProxyUserCreationAllowed(),
		"BASE_PATH":                          o.BasePath(),
		"BASE_URL":                           o.BaseURL(),
		"BATCH_SIZE":                         o.BatchSize(),
		"CERT_DOMAIN":                        o.CertDomain(),
		"CERT_FILE":                          o.CertFile(),
		"CLEANUP_ARCHIVE_BATCH_SIZE":         o.CleanupArchiveBatchSize(),
		"CLEANUP_ARCHIVE_READ_DAYS":          o.CleanupArchiveReadDays(),
		"CLEANUP_ARCHIVE_UNREAD_DAYS":        o.CleanupArchiveUnreadDays(),
		"CLEANUP_FREQUENCY_HOURS":            o.env.CleanupFrequencyHours,
		"CLEANUP_REMOVE_SESSIONS_DAYS":       o.CleanupRemoveSessionsDays(),
		"CLEANUP_INACTIVE_SESSIONS_DAYS":     o.CleanupInactiveSessionsDays(),
		"CONNECTIONS_PER_SERVER":             o.ConnectionsPerServer(),
		"CREATE_ADMIN":                       o.CreateAdmin(),
		"DATABASE_CONNECTION_LIFETIME":       o.env.DatabaseConnectionLifetime,
		"DATABASE_MAX_CONNS":                 o.DatabaseMaxConns(),
		"DATABASE_MIN_CONNS":                 o.DatabaseMinConns(),
		"DATABASE_URL":                       secretValue(o.DatabaseURL(), redactSecret),
		"DISABLE_HSTS":                       !o.HasHSTS(),
		"DISABLE_HTTP_SERVICE":               !o.HasHTTPService(),
		"DISABLE_SCHEDULER_SERVICE":          !o.HasSchedulerService(),
		"FILTER_ENTRY_MAX_AGE_DAYS":          o.FilterEntryMaxAgeDays(),
		"FETCH_YOUTUBE_WATCH_TIME":           o.FetchYouTubeWatchTime(),
		"FETCH_NEBULA_WATCH_TIME":            o.FetchNebulaWatchTime(),
		"FETCH_ODYSEE_WATCH_TIME":            o.FetchOdyseeWatchTime(),
		"FETCH_BILIBILI_WATCH_TIME":          o.FetchBilibiliWatchTime(),
		"HTTPS":                              o.HasHSTS(),
		"HTTP_CLIENT_MAX_BODY_SIZE":          o.HTTPClientMaxBodySize(),
		"HTTP_CLIENT_PROXIES":                clientProxyURLsRedacted,
		"HTTP_CLIENT_PROXY":                  clientProxyURLRedacted,
		"HTTP_CLIENT_TIMEOUT":                o.env.HttpClientTimeout,
		"HTTP_CLIENT_USER_AGENT":             o.HTTPClientUserAgent(),
		"HTTP_SERVER_TIMEOUT":                o.env.HttpServerTimeout,
		"HTTP_SERVICE":                       o.HasHTTPService(),
		"INVIDIOUS_INSTANCE":                 o.InvidiousInstance(),
		"KEY_FILE":                           o.CertKeyFile(),
		"LISTEN_ADDR":                        o.ListenAddr(),
		"LOG_FILE":                           o.LogFile(),
		"LOG_DATE_TIME":                      o.LogDateTime(),
		"LOG_FORMAT":                         o.LogFormat(),
		"LOG_LEVEL":                          o.LogLevel(),
		"MAINTENANCE_MESSAGE":                o.MaintenanceMessage(),
		"MAINTENANCE_MODE":                   o.HasMaintenanceMode(),
		"METRICS_ALLOWED_NETWORKS":           strings.Join(o.MetricsAllowedNetworks(), ","),
		"METRICS_COLLECTOR":                  o.HasMetricsCollector(),
		"METRICS_PASSWORD":                   secretValue(o.MetricsPassword(), redactSecret),
		"METRICS_REFRESH_INTERVAL":           o.env.MetricsRefreshInterval,
		"METRICS_USERNAME":                   o.MetricsUsername(),
		"OAUTH2_CLIENT_ID":                   o.OAuth2ClientID(),
		"OAUTH2_CLIENT_SECRET":               secretValue(o.OAuth2ClientSecret(), redactSecret),
		"OAUTH2_OIDC_DISCOVERY_ENDPOINT":     o.OIDCDiscoveryEndpoint(),
		"OAUTH2_OIDC_PROVIDER_NAME":          o.OIDCProviderName(),
		"OAUTH2_PROVIDER":                    o.OAuth2Provider(),
		"OAUTH2_REDIRECT_URL":                o.OAuth2RedirectURL(),
		"OAUTH2_USER_CREATION":               o.IsOAuth2UserCreationAllowed(),
		"DISABLE_LOCAL_AUTH":                 o.DisableLocalAuth(),
		"POLLING_FREQUENCY":                  o.env.PollingFrequency,
		"FORCE_REFRESH_INTERVAL":             o.ForceRefreshInterval(),
		"POLLING_PARSING_ERROR_LIMIT":        o.PollingParsingErrorLimit(),
		"MEDIA_PROXY_HTTP_CLIENT_TIMEOUT":    o.env.MediaProxyHTTPClientTimeout,
		"MEDIA_PROXY_RESOURCE_TYPES":         strings.Join(o.MediaProxyResourceTypes(), ","),
		"MEDIA_PROXY_MODE":                   o.MediaProxyMode(),
		"MEDIA_PROXY_PRIVATE_KEY":            mediaProxyPrivateKeyValue,
		"MEDIA_PROXY_CUSTOM_URL":             o.MediaCustomProxyURL(),
		"ROOT_URL":                           o.RootURL(),
		"RUN_MIGRATIONS":                     o.RunMigrations(),
		"SCHEDULER_ROUND_ROBIN_MIN_INTERVAL": o.SchedulerRoundRobinMinInterval(),
		"SCHEDULER_ROUND_ROBIN_MAX_INTERVAL": o.SchedulerRoundRobinMaxInterval(),
		"SCHEDULER_SERVICE":                  !o.HasSchedulerService(),
		"WATCHDOG":                           o.HasWatchdog(),
		"WORKER_POOL_SIZE":                   o.WorkerPoolSize(),
		"YOUTUBE_API_KEY":                    secretValue(o.YouTubeApiKey(), redactSecret),
		"YOUTUBE_EMBED_URL_OVERRIDE":         o.YouTubeEmbedUrlOverride(),
		"WEBAUTHN":                           o.WebAuthn(),
		"PREFER_SITE_ICON":                   o.PreferSiteIcon(),
		"RATE_LIMIT_PER_SERVER":              o.RateLimitPerServer(),
		"TRUSTED_PROXIES":                    strings.Join(o.env.TrustedProxies, ","),
	}

	sortedKeys := slices.Sorted(maps.Keys(keyValues))
	sortedOptions := make([]Option, len(sortedKeys))
	for i, key := range sortedKeys {
		sortedOptions[i] = Option{Key: key, Value: keyValues[key]}
	}
	return sortedOptions
}

func (o *Options) String() string {
	var builder strings.Builder
	for _, option := range o.SortedOptions(false) {
		fmt.Fprintf(&builder, "%s=%v\n", option.Key, option.Value)
	}
	return builder.String()
}

func secretValue(value string, redactSecret bool) string {
	if redactSecret && value != "" {
		return "<secret>"
	}
	return value
}
