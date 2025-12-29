// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config // import "miniflux.app/v2/internal/config"

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/url"
	"runtime"
	"slices"
	"strings"
	"time"

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

// options contains configuration options.
type options struct {
	HostLimits map[string]HostLimits `yaml:"host_limits" validate:"dive,keys,required,endkeys,required"`

	env envOptions

	root                 *url.URL
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

type envOptions struct {
	DisableAPI                     bool     `env:"DISABLE_API"`
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
	ConnectionsPerServer           int64    `env:"CONNECTIONS_PER_SERVER" validate:"min=0"`
	RateLimitPerServer             float64  `env:"RATE_LIMIT_PER_SERVER" validate:"min=0"`
	TrustedProxies                 []string `env:"TRUSTED_PROXIES" validate:"dive,required,ip"`
	Testing                        bool     `env:"TESTING"`
	Operators                      []string `env:"OPERATORS"`

	PollingErrorLimit int           `env:"POLLING_PARSING_ERROR_LIMIT" validate:"min=0"`
	PollingErrorRetry time.Duration `env:"POLLING_ERROR_RETRY" validate:"min=0"`
}

type Log struct {
	LogFile     string `env:"FILE" validate:"required"`
	LogDateTime bool   `env:"DATE_TIME"`
	LogFormat   string `env:"FORMAT" validate:"required,oneof=human json text"`
	LogLevel    string `env:"LEVEL" validate:"required,oneof=debug info warning error"`
}

// NewOptions returns Options with default values.
func NewOptions() *options {
	maxConns := max(4, runtime.GOMAXPROCS(0))

	return &options{
		HostLimits: map[string]HostLimits{},

		env: envOptions{
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
			PollingErrorLimit:              3,
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
			ConnectionsPerServer:           8,
			RateLimitPerServer:             10,
			TrustedProxies:                 []string{"127.0.0.1"},
		},

		rootURL: defaultBaseURL,
	}
}

func (o *options) init() (err error) {
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

	o.env.BaseURL, o.root, err = parseBaseURL(o.env.BaseURL)
	if err != nil {
		return err
	}

	o.basePath = o.root.EscapedPath()
	o.root.Path = ""
	o.rootURL = o.root.String()
	return nil
}

func (o *options) validate() error {
	if err := Validator().Struct(&o.env); err != nil {
		return fmt.Errorf("config: failed validate: %w", err)
	}

	if o.env.DisableLocalAuth {
		switch {
		case o.env.Oauth2Provider == "" && o.env.AuthProxyHeader == "":
			return errors.New("DISABLE_LOCAL_AUTH is enabled but neither OAUTH2_PROVIDER nor AUTH_PROXY_HEADER is not set. Please enable at least one authentication source")
		case o.env.Oauth2Provider != "" && !o.env.Oauth2UserCreationAllowed:
			return errors.New("DISABLE_LOCAL_AUTH is enabled and an OAUTH2_PROVIDER is configured, but OAUTH2_USER_CREATION is not enabled")
		case o.env.AuthProxyHeader != "" && !o.env.AuthProxyUserCreation:
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

func (o *options) applyFileStrings() {
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

func (o *options) applyPrivateKeys() error {
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

func (o *options) makeTrustedProxies() {
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

func parseBaseURL(value string) (string, *url.URL, error) {
	if value == "" {
		value = defaultBaseURL
	}

	if value[len(value)-1:] == "/" {
		value = value[:len(value)-1]
	}

	u, err := url.Parse(value)
	if err != nil {
		return "", nil, fmt.Errorf("config: invalid BASE_URL: %w", err)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "https" && scheme != "http" {
		return "", nil, errors.New(
			"config: invalid BASE_URL: scheme must be http or https")
	}
	return value, u, nil
}

func (o *options) sortedOptions(redactSecret bool) []Option {
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
		"ADMIN_PASSWORD":                     secretValue(o.env.AdminPassword, redactSecret),
		"ADMIN_USERNAME":                     o.env.AdminUsername,
		"AUTH_PROXY_HEADER":                  o.env.AuthProxyHeader,
		"AUTH_PROXY_USER_CREATION":           o.env.AuthProxyUserCreation,
		"BASE_PATH":                          o.basePath,
		"BASE_URL":                           o.env.BaseURL,
		"BATCH_SIZE":                         o.env.BatchSize,
		"CERT_DOMAIN":                        o.env.CertDomain,
		"CERT_FILE":                          o.env.CertFile,
		"CLEANUP_ARCHIVE_BATCH_SIZE":         o.env.CleanupArchiveBatchSize,
		"CLEANUP_ARCHIVE_READ_DAYS":          o.env.CleanupArchiveReadDays,
		"CLEANUP_ARCHIVE_UNREAD_DAYS":        o.env.CleanupArchiveUnreadDays,
		"CLEANUP_FREQUENCY_HOURS":            o.env.CleanupFrequencyHours,
		"CLEANUP_REMOVE_SESSIONS_DAYS":       o.env.CleanupRemoveSessionsDays,
		"CLEANUP_INACTIVE_SESSIONS_DAYS":     o.env.CleanupInactiveSessionsDays,
		"CONNECTIONS_PER_SERVER":             o.env.ConnectionsPerServer,
		"CREATE_ADMIN":                       o.env.CreateAdmin,
		"DATABASE_CONNECTION_LIFETIME":       o.env.DatabaseConnectionLifetime,
		"DATABASE_MAX_CONNS":                 o.env.DatabaseMaxConns,
		"DATABASE_MIN_CONNS":                 o.env.DatabaseMinConns,
		"DATABASE_URL":                       secretValue(o.env.DatabaseURL, redactSecret),
		"DISABLE_API":                        o.env.DisableAPI,
		"DISABLE_HSTS":                       o.env.DisableHSTS,
		"DISABLE_HTTP_SERVICE":               o.env.DisableHttpService,
		"DISABLE_SCHEDULER_SERVICE":          o.env.DisableScheduler,
		"FILTER_ENTRY_MAX_AGE_DAYS":          o.env.FilterEntryMaxAgeDays,
		"FETCH_YOUTUBE_WATCH_TIME":           o.env.FetchYouTubeWatchTime,
		"FETCH_NEBULA_WATCH_TIME":            o.env.FetchNebulaWatchTime,
		"FETCH_ODYSEE_WATCH_TIME":            o.env.FetchOdyseeWatchTime,
		"FETCH_BILIBILI_WATCH_TIME":          o.env.FetchBilibiliWatchTime,
		"HTTPS":                              !o.env.DisableHSTS,
		"HTTP_CLIENT_MAX_BODY_SIZE":          o.env.HttpClientMaxBodySize,
		"HTTP_CLIENT_PROXIES":                clientProxyURLsRedacted,
		"HTTP_CLIENT_PROXY":                  clientProxyURLRedacted,
		"HTTP_CLIENT_TIMEOUT":                o.env.HttpClientTimeout,
		"HTTP_CLIENT_USER_AGENT":             o.env.HttpClientUserAgent,
		"HTTP_SERVER_TIMEOUT":                o.env.HttpServerTimeout,
		"HTTP_SERVICE":                       !o.env.DisableHttpService,
		"INVIDIOUS_INSTANCE":                 o.env.InvidiousInstance,
		"KEY_FILE":                           o.env.CertKeyFile,
		"LISTEN_ADDR":                        o.env.ListenAddr,
		"LOG_FILE":                           o.env.LogFile,
		"LOG_DATE_TIME":                      o.env.LogDateTime,
		"LOG_FORMAT":                         o.env.LogFormat,
		"LOG_LEVEL":                          o.env.LogLevel,
		"MAINTENANCE_MESSAGE":                o.env.MaintenanceMessage,
		"MAINTENANCE_MODE":                   o.env.MaintenanceMode,
		"METRICS_ALLOWED_NETWORKS":           strings.Join(o.env.MetricsAllowedNetworks, ","),
		"METRICS_COLLECTOR":                  o.env.MetricsCollector,
		"METRICS_PASSWORD":                   secretValue(o.env.MetricsPassword, redactSecret),
		"METRICS_REFRESH_INTERVAL":           o.env.MetricsRefreshInterval,
		"METRICS_USERNAME":                   o.env.MetricsUsername,
		"OAUTH2_CLIENT_ID":                   o.env.Oauth2ClientID,
		"OAUTH2_CLIENT_SECRET":               secretValue(o.env.Oauth2ClientSecret, redactSecret),
		"OAUTH2_OIDC_DISCOVERY_ENDPOINT":     o.env.OidcDiscoveryEndpoint,
		"OAUTH2_OIDC_PROVIDER_NAME":          o.env.OidcProviderName,
		"OAUTH2_PROVIDER":                    o.env.Oauth2Provider,
		"OAUTH2_REDIRECT_URL":                o.env.Oauth2RedirectURL,
		"OAUTH2_USER_CREATION":               o.env.Oauth2UserCreationAllowed,
		"DISABLE_LOCAL_AUTH":                 o.env.DisableLocalAuth,
		"POLLING_FREQUENCY":                  o.env.PollingFrequency,
		"FORCE_REFRESH_INTERVAL":             o.env.ForceRefreshInterval,
		"POLLING_PARSING_ERROR_LIMIT":        o.env.PollingErrorLimit,
		"MEDIA_PROXY_HTTP_CLIENT_TIMEOUT":    o.env.MediaProxyHTTPClientTimeout,
		"MEDIA_PROXY_RESOURCE_TYPES":         strings.Join(o.env.MediaProxyResourceTypes, ","),
		"MEDIA_PROXY_MODE":                   o.env.MediaProxyMode,
		"MEDIA_PROXY_PRIVATE_KEY":            mediaProxyPrivateKeyValue,
		"MEDIA_PROXY_CUSTOM_URL":             o.env.MediaProxyCustomURL,
		"ROOT_URL":                           o.rootURL,
		"RUN_MIGRATIONS":                     o.env.RunMigrations,
		"SCHEDULER_ROUND_ROBIN_MIN_INTERVAL": o.env.SchedulerRoundRobinMinInterval,
		"SCHEDULER_ROUND_ROBIN_MAX_INTERVAL": o.env.SchedulerRoundRobinMaxInterval,
		"SCHEDULER_SERVICE":                  !o.env.DisableScheduler,
		"WATCHDOG":                           o.env.Watchdog,
		"WORKER_POOL_SIZE":                   o.env.WorkerPoolSize,
		"YOUTUBE_API_KEY":                    secretValue(o.env.YouTubeApiKey, redactSecret),
		"YOUTUBE_EMBED_URL_OVERRIDE":         o.env.YouTubeEmbedUrlOverride.String(),
		"WEBAUTHN":                           o.env.WebAuthn,
		"PREFER_SITE_ICON":                   o.env.PreferSiteIcon,
		"RATE_LIMIT_PER_SERVER":              o.env.RateLimitPerServer,
		"TRUSTED_PROXIES":                    strings.Join(o.env.TrustedProxies, ","),
	}

	sortedKeys := slices.Sorted(maps.Keys(keyValues))
	sortedOptions := make([]Option, len(sortedKeys))
	for i, key := range sortedKeys {
		sortedOptions[i] = Option{Key: key, Value: keyValues[key]}
	}
	return sortedOptions
}

func secretValue(value string, redactSecret bool) string {
	if redactSecret && value != "" {
		return "<secret>"
	}
	return value
}

func (o *options) String() string {
	var builder strings.Builder
	for _, option := range o.sortedOptions(false) {
		fmt.Fprintf(&builder, "%s=%v\n", option.Key, option.Value)
	}
	return builder.String()
}

func HTTPS() bool  { return opts.env.HTTPS }
func EnableHTTPS() { opts.env.HTTPS = true }

func LogFile() string { return opts.env.LogFile }

// LogDateTime returns true if the date/time should be displayed in log
// messages.
func LogDateTime() bool { return opts.env.LogDateTime }

// LogFormat returns the log format.
func LogFormat() string { return opts.env.LogFormat }

// LogLevel returns the log level.
func LogLevel() string { return opts.env.LogLevel }

// SetLogLevel sets the log level.
func SetLogLevel(level string) { opts.env.LogLevel = level }

// HasMaintenanceMode returns true if maintenance mode is enabled.
func HasMaintenanceMode() bool { return opts.env.MaintenanceMode }

// MaintenanceMessage returns maintenance message.
func MaintenanceMessage() string { return opts.env.MaintenanceMessage }

// BaseURL returns the application base URL with path.
func BaseURL() string { return opts.env.BaseURL }

// RootURL returns the base URL without path.
func RootURL() string { return opts.rootURL }

// BasePath returns the application base path according to the base URL.
func BasePath() string { return opts.basePath }

func Root() *url.URL { return opts.root }

// IsDefaultDatabaseURL returns true if the default database URL is used.
func IsDefaultDatabaseURL() bool {
	return opts.env.DatabaseURL == defaultDatabaseURL
}

// DatabaseURL returns the database URL.
func DatabaseURL() string { return opts.env.DatabaseURL }

// DatabaseMaxConns returns the maximum number of database connections.
func DatabaseMaxConns() int { return opts.env.DatabaseMaxConns }

// DatabaseMinConns returns the minimum number of database connections.
func DatabaseMinConns() int { return opts.env.DatabaseMinConns }

// DatabaseConnectionLifetime returns the maximum amount of time a connection
// may be reused.
func DatabaseConnectionLifetime() time.Duration {
	return time.Duration(opts.env.DatabaseConnectionLifetime) * time.Minute
}

// ListenAddr returns the listen address for the HTTP server.
func ListenAddr() string { return opts.env.ListenAddr }

// CertFile returns the SSL certificate filename if any.
func CertFile() string { return opts.env.CertFile }

// CertKeyFile returns the private key filename for custom SSL certificate.
func CertKeyFile() string { return opts.env.CertKeyFile }

// CertDomain returns the domain to use for Let's Encrypt certificate.
func CertDomain() string { return opts.env.CertDomain }

// CleanupFrequencyHours returns the interval in hours for cleanup jobs.
func CleanupFrequencyHours() time.Duration {
	return time.Duration(opts.env.CleanupFrequencyHours) * time.Hour
}

// CleanupArchiveReadDays returns the number of days after which marking read
// items as removed.
func CleanupArchiveReadDays() int { return opts.env.CleanupArchiveReadDays }

// CleanupArchiveUnreadDays returns the number of days after which marking
// unread items as removed.
func CleanupArchiveUnreadDays() int { return opts.env.CleanupArchiveUnreadDays }

// CleanupArchiveBatchSize returns the number of entries to archive for each
// interval.
func CleanupArchiveBatchSize() int { return opts.env.CleanupArchiveBatchSize }

// CleanupRemoveSessionsDays returns the number of days after which to remove
// sessions.
func CleanupRemoveSessionsDays() int {
	return opts.env.CleanupRemoveSessionsDays
}

func CleanupRemoveSessionsInterval() time.Duration {
	return time.Duration(opts.env.CleanupRemoveSessionsDays) * 24 * time.Hour
}

func CleanupInactiveSessionsDays() int {
	return opts.env.CleanupInactiveSessionsDays
}

func CleanupInactiveSessionsInterval() time.Duration {
	return time.Duration(opts.env.CleanupInactiveSessionsDays) * 24 * time.Hour
}

// WorkerPoolSize returns the number of background worker.
func WorkerPoolSize() int { return opts.env.WorkerPoolSize }

// PollingFrequency returns the interval to refresh feeds in the background.
func PollingFrequency() time.Duration {
	return time.Duration(opts.env.PollingFrequency) * time.Minute
}

// ForceRefreshInterval returns the force refresh interval
func ForceRefreshInterval() int { return opts.env.ForceRefreshInterval }

// BatchSize returns the number of feeds to send for background processing.
func BatchSize() int { return opts.env.BatchSize }

func SchedulerRoundRobinMinInterval() int {
	return opts.env.SchedulerRoundRobinMinInterval
}

func SchedulerRoundRobinMaxInterval() int {
	return opts.env.SchedulerRoundRobinMaxInterval
}

// PollingErrorLimit returns the limit of errors when to stop polling.
func PollingErrorLimit() int { return opts.env.PollingErrorLimit }

func PollingErrorLimited(count int) bool {
	return count >= opts.env.PollingErrorLimit
}

func PollingErrorRetry() time.Duration { return opts.env.PollingErrorRetry }

// IsOAuth2UserCreationAllowed returns true if user creation is allowed for
// OAuth2 users.
func IsOAuth2UserCreationAllowed() bool {
	return opts.env.Oauth2UserCreationAllowed
}

// OAuth2ClientID returns the OAuth2 Client ID.
func OAuth2ClientID() string { return opts.env.Oauth2ClientID }

// OAuth2ClientSecret returns the OAuth2 client secret.
func OAuth2ClientSecret() string { return opts.env.Oauth2ClientSecret }

// OAuth2RedirectURL returns the OAuth2 redirect URL.
func OAuth2RedirectURL() string { return opts.env.Oauth2RedirectURL }

// OIDCDiscoveryEndpoint returns the OAuth2 OIDC discovery endpoint.
func OIDCDiscoveryEndpoint() string { return opts.env.OidcDiscoveryEndpoint }

// OIDCProviderName returns the OAuth2 OIDC provider's display name
func OIDCProviderName() string { return opts.env.OidcProviderName }

// OAuth2Provider returns the name of the OAuth2 provider configured.
func OAuth2Provider() string { return opts.env.Oauth2Provider }

// DisableLocalAUth returns true if the local user database should not be used
// to authenticate users.
func DisableLocalAuth() bool { return opts.env.DisableLocalAuth }

func HasAPI() bool { return !opts.env.DisableAPI }

// HasHSTS returns true if HTTP Strict Transport Security is enabled.
func HasHSTS() bool { return !opts.env.DisableHSTS }

// RunMigrations returns true if the environment variable RUN_MIGRATIONS is not
// empty.
func RunMigrations() bool { return opts.env.RunMigrations }

// CreateAdmin returns true if the environment variable CREATE_ADMIN is not
// empty.
func CreateAdmin() bool { return opts.env.CreateAdmin }

// AdminUsername returns the admin username if defined.
func AdminUsername() string { return opts.env.AdminUsername }

// AdminPassword returns the admin password if defined.
func AdminPassword() string { return opts.env.AdminPassword }

// FetchYouTubeWatchTime returns true if the YouTube video duration should be
// fetched and used as a reading time.
func FetchYouTubeWatchTime() bool { return opts.env.FetchYouTubeWatchTime }

// YouTubeApiKey returns the YouTube API key if defined.
func YouTubeApiKey() string { return opts.env.YouTubeApiKey }

// YouTubeEmbedUrlOverride returns the YouTube embed URL override if defined.
func YouTubeEmbedUrlOverride() string { return YouTubeEmbedURL().String() }

// YouTubeEmbedDomain returns the domain used for YouTube embeds.
func YouTubeEmbedDomain() string { return YouTubeEmbedURL().Hostname() }

func YouTubeEmbedURL() *url.URL { return opts.env.YouTubeEmbedUrlOverride }

// FetchNebulaWatchTime returns true if the Nebula video duration should be
// fetched and used as a reading time.
func FetchNebulaWatchTime() bool { return opts.env.FetchNebulaWatchTime }

// FetchOdyseeWatchTime returns true if the Odysee video duration should be
// fetched and used as a reading time.
func FetchOdyseeWatchTime() bool { return opts.env.FetchOdyseeWatchTime }

// FetchBilibiliWatchTime returns true if the Bilibili video duration should be
// fetched and used as a reading time.
func FetchBilibiliWatchTime() bool { return opts.env.FetchBilibiliWatchTime }

// MediaProxyMode returns "none" to never proxy, "http-only" to proxy non-HTTPS,
// "all" to always proxy.
func MediaProxyMode() string { return opts.env.MediaProxyMode }

// MediaProxyResourceTypes returns a slice of resource types to proxy.
func MediaProxyResourceTypes() []string {
	return opts.env.MediaProxyResourceTypes
}

// MediaCustomProxyURL returns the custom proxy URL for medias.
func MediaCustomProxyURL() *url.URL { return opts.env.MediaProxyCustomURL }

// MediaProxyHTTPClientTimeout returns the time limit in seconds before the
// proxy HTTP client cancel the request.
func MediaProxyHTTPClientTimeout() time.Duration {
	return time.Duration(opts.env.MediaProxyHTTPClientTimeout) * time.Minute
}

// MediaProxyPrivateKey returns the private key used by the media proxy.
func MediaProxyPrivateKey() []byte { return opts.mediaProxyPrivateKey }

// HasHTTPService returns true if the HTTP service is enabled.
func HasHTTPService() bool { return !opts.env.DisableHttpService }

// HasSchedulerService returns true if the scheduler service is enabled.
func HasSchedulerService() bool { return !opts.env.DisableScheduler }

// HTTPClientTimeout returns the time limit in seconds before the HTTP client
// cancel the request.
func HTTPClientTimeout() time.Duration {
	return time.Duration(opts.env.HttpClientTimeout) * time.Second
}

// HTTPClientMaxBodySize returns the number of bytes allowed for the HTTP client
// to transfer.
func HTTPClientMaxBodySize() int64 { return opts.env.HttpClientMaxBodySize }

// HTTPClientProxyURL returns the client HTTP proxy URL if configured.
func HTTPClientProxyURL() *url.URL { return opts.env.HttpClientProxyURL }

// HasHTTPClientProxyURLConfigured returns true if the client HTTP proxy URL if
// configured.
func HasHTTPClientProxyURLConfigured() bool {
	return opts.env.HttpClientProxyURL != nil
}

// HTTPClientProxies returns the list of proxies.
func HTTPClientProxies() []string { return opts.env.HttpClientProxies }

// HTTPClientProxiesString returns true if the list of rotating proxies are
// configured.
func HasHTTPClientProxiesConfigured() bool {
	return len(opts.env.HttpClientProxies) > 0
}

// HTTPServerTimeout returns the time limit in seconds before the HTTP server
// cancel the request.
func HTTPServerTimeout() time.Duration {
	return time.Duration(opts.env.HttpServerTimeout) * time.Second
}

// AuthProxyHeader returns an HTTP header name that contains username for
// authentication using auth proxy.
func AuthProxyHeader() string { return opts.env.AuthProxyHeader }

// IsAuthProxyUserCreationAllowed returns true if user creation is allowed for
// users authenticated using auth proxy.
func IsAuthProxyUserCreationAllowed() bool {
	return opts.env.AuthProxyUserCreation
}

// HasMetricsCollector returns true if metrics collection is enabled.
func HasMetricsCollector() bool { return opts.env.MetricsCollector }

// MetricsRefreshInterval returns the refresh interval.
func MetricsRefreshInterval() time.Duration {
	return time.Duration(opts.env.MetricsRefreshInterval) * time.Second
}

// MetricsAllowedNetworks returns the list of networks allowed to connect to the
// metrics endpoint.
func MetricsAllowedNetworks() []string {
	return opts.env.MetricsAllowedNetworks
}

func MetricsUsername() string { return opts.env.MetricsUsername }
func MetricsPassword() string { return opts.env.MetricsPassword }

// HTTPClientUserAgent returns the global User-Agent header for miniflux.
func HTTPClientUserAgent() string { return opts.env.HttpClientUserAgent }

// HasWatchdog returns true if the systemd watchdog is enabled.
func HasWatchdog() bool { return opts.env.Watchdog }

// InvidiousInstance returns the invidious instance used by miniflux
func InvidiousInstance() string { return opts.env.InvidiousInstance }

// WebAuthn returns true if WebAuthn logins are supported
func WebAuthn() bool { return opts.env.WebAuthn }

// FilterEntryMaxAgeDays returns the number of days after which entries should
// be retained.
func FilterEntryMaxAgeDays() int { return opts.env.FilterEntryMaxAgeDays }

func FilterEntryMaxAge() time.Duration {
	return time.Duration(opts.env.FilterEntryMaxAgeDays) * 24 * time.Hour
}

func PreferSiteIcon() bool { return opts.env.PreferSiteIcon }

func ConnectionsPerServer() int64 { return opts.env.ConnectionsPerServer }

func RateLimitPerServer() float64 { return opts.env.RateLimitPerServer }

func TrustedProxy(ip string) bool {
	_, ok := opts.trustedProxies[ip]
	return ok
}

func Operator(username string) bool {
	if len(opts.env.Operators) == 0 {
		return false
	}
	return slices.Contains(opts.env.Operators, username)
}

func Logging() []Log {
	if len(opts.env.Logging) == 0 {
		return []Log{{
			LogFile:     opts.env.LogFile,
			LogDateTime: opts.env.LogDateTime,
			LogFormat:   opts.env.LogFormat,
			LogLevel:    opts.env.LogLevel,
		}}
	}
	return slices.Clone(opts.env.Logging)
}

func FindHostLimits(hostname string) (found HostLimits) {
	for hostname != "" {
		if limits, ok := opts.HostLimits[hostname]; ok {
			found = limits
			break
		}
		_, hostname, _ = strings.Cut(hostname, ".")
	}
	return found.withDefaults(opts.env.ConnectionsPerServer,
		opts.env.RateLimitPerServer)
}

// SortedOptions returns options as a list of key value pairs, sorted by keys.
func SortedOptions(redactSecret bool) []Option {
	return opts.sortedOptions(redactSecret)
}

func String() string { return opts.String() }
