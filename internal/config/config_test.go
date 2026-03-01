// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config // import "miniflux.app/v2/internal/config"

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogFileDefaultValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.LogFile, opts.env.LogFile)
}

func parseEnvironmentVariables(t *testing.T) *options {
	t.Helper()
	parser := NewParser()
	opts, err := parser.ParseEnvironmentVariables()
	require.NoError(t, err)
	require.NotNil(t, opts)
	return opts
}

func TestLogFileWithCustomFilename(t *testing.T) {
	os.Clearenv()
	const want = "foobar.log"
	t.Setenv("LOG_FILE", want)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, want, opts.env.LogFile)
}

func TestLogFileWithEmptyValue(t *testing.T) {
	os.Clearenv()
	t.Setenv("LOG_FILE", "")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.LogFile, opts.env.LogFile)
}

func TestLogLevelDefaultValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.LogLevel, opts.env.LogLevel)
}

func TestLogLevelWithCustomValue(t *testing.T) {
	os.Clearenv()
	const want = "warning"
	t.Setenv("LOG_LEVEL", want)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, want, opts.env.LogLevel)
}

func TestLogLevelWithInvalidValue(t *testing.T) {
	os.Clearenv()
	t.Setenv("LOG_LEVEL", "invalid")
	_, err := NewParser().ParseEnvironmentVariables()
	require.ErrorContains(t, err, "oneof")
}

func TestLogDateTimeDefaultValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.LogDateTime, opts.env.LogDateTime)
}

func TestLogDateTimeWithCustomValue(t *testing.T) {
	os.Clearenv()
	t.Setenv("LOG_DATE_TIME", "true")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.env.LogDateTime)
}

func TestLogDateTimeWithInvalidValue(t *testing.T) {
	os.Clearenv()
	t.Setenv("LOG_DATE_TIME", "invalid")
	_, err := NewParser().ParseEnvironmentVariables()
	t.Log(err)
	require.ErrorContains(t, err, "invalid syntax")
}

func TestLogFormatDefaultValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.LogFormat, opts.env.LogFormat)
}

func TestLogFormatWithCustomValue(t *testing.T) {
	os.Clearenv()
	const want = "json"
	t.Setenv("LOG_FORMAT", want)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, want, opts.env.LogFormat)
}

func TestLogFormatWithInvalidValue(t *testing.T) {
	os.Clearenv()
	t.Setenv("LOG_FORMAT", "invalid")
	_, err := NewParser().ParseEnvironmentVariables()
	t.Log(err)
	require.ErrorContains(t, err, "failed on the 'oneof' tag")
}

func TestCustomBaseURL(t *testing.T) {
	os.Clearenv()
	const want = "http://example.org"
	t.Setenv("BASE_URL", want)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, want, opts.env.BaseURL)
	assert.Equal(t, want, opts.rootURL)
	assert.Empty(t, opts.basePath)
}

func TestCustomBaseURLWithTrailingSlash(t *testing.T) {
	os.Clearenv()
	t.Setenv("BASE_URL", "http://example.org/folder/")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, "http://example.org/folder", opts.env.BaseURL)
	assert.Equal(t, "http://example.org", opts.rootURL)
	assert.Equal(t, "/folder", opts.basePath)
}

func TestCustomBaseURLWithCustomPort(t *testing.T) {
	os.Clearenv()
	t.Setenv("BASE_URL", "http://example.org:88/folder/")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, "http://example.org:88/folder", opts.env.BaseURL)
	assert.Equal(t, "http://example.org:88", opts.rootURL)
	assert.Equal(t, "/folder", opts.basePath)
}

func TestBaseURLWithoutScheme(t *testing.T) {
	os.Clearenv()
	t.Setenv("BASE_URL", "example.org/folder/")
	_, err := NewParser().ParseEnvironmentVariables()
	require.Error(t, err)
}

func TestBaseURLWithInvalidScheme(t *testing.T) {
	os.Clearenv()
	t.Setenv("BASE_URL", "ftp://example.org/folder/")
	_, err := NewParser().ParseEnvironmentVariables()
	require.Error(t, err)
}

func TestInvalidBaseURL(t *testing.T) {
	os.Clearenv()
	t.Setenv("BASE_URL", "http://example|org")
	_, err := NewParser().ParseEnvironmentVariables()
	require.Error(t, err)
}

func TestDefaultBaseURL(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	defs := NewOptions()
	assert.Equal(t, defs.env.BaseURL, opts.env.BaseURL)
	assert.Equal(t, defs.rootURL, opts.rootURL)
	assert.Empty(t, opts.basePath)
}

func TestDatabaseURL(t *testing.T) {
	os.Clearenv()
	const expected = "foobar"
	t.Setenv("DATABASE_URL", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.env.DatabaseURL)
	assert.NotEqual(t, defaultDatabaseURL, opts.env.DatabaseURL)
}

func TestDefaultDatabaseURLValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.DatabaseURL, opts.env.DatabaseURL)
	assert.Equal(t, defaultDatabaseURL, opts.env.DatabaseURL)
}

func TestDefaultDatabaseMaxConnsValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.DatabaseMaxConns, opts.env.DatabaseMaxConns)
}

func TestDatabaseMaxConns(t *testing.T) {
	os.Clearenv()
	t.Setenv("DATABASE_MAX_CONNS", "42")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 42, opts.env.DatabaseMaxConns)
}

func TestDefaultDatabaseMinConnsValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.DatabaseMinConns, opts.env.DatabaseMinConns)
}

func TestDatabaseMinConns(t *testing.T) {
	os.Clearenv()
	t.Setenv("DATABASE_MIN_CONNS", "42")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 42, opts.env.DatabaseMinConns)
}

func TestListenAddr(t *testing.T) {
	os.Clearenv()
	const expected = "foobar"
	t.Setenv("LISTEN_ADDR", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.env.ListenAddr)
}

func TestListenAddrWithPortDefined(t *testing.T) {
	os.Clearenv()
	t.Setenv("PORT", "3000")
	t.Setenv("LISTEN_ADDR", "foobar")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, ":3000", opts.env.ListenAddr)
}

func TestDefaultListenAddrValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.ListenAddr, opts.env.ListenAddr)
}

func TestCertFile(t *testing.T) {
	os.Clearenv()
	const expected = "foobar"
	t.Setenv("CERT_FILE", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.env.CertFile)
}

func TestDefaultCertFileValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.CertFile, opts.env.CertFile)
}

func TestKeyFile(t *testing.T) {
	os.Clearenv()
	const expected = "foobar"
	t.Setenv("KEY_FILE", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.env.CertKeyFile)
}

func TestDefaultKeyFileValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.CertKeyFile, opts.env.CertKeyFile)
}

func TestCertDomain(t *testing.T) {
	os.Clearenv()
	const expected = "example.org"
	t.Setenv("CERT_DOMAIN", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.env.CertDomain)
}

func TestDefaultCertDomainValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.CertDomain, opts.env.CertDomain)
}

func TestDefaultCleanupFrequencyHoursValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.CleanupFrequencyHours,
		opts.env.CleanupFrequencyHours)
}

func TestCleanupFrequencyHours(t *testing.T) {
	os.Clearenv()
	t.Setenv("CLEANUP_FREQUENCY_HOURS", "42")
	t.Setenv("CLEANUP_FREQUENCY", "19")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 42, opts.env.CleanupFrequencyHours)
}

func TestDefaultCleanupArchiveReadDaysValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 60, opts.env.CleanupArchiveReadDays)
}

func TestCleanupArchiveReadDays(t *testing.T) {
	os.Clearenv()
	t.Setenv("CLEANUP_ARCHIVE_READ_DAYS", "7")
	t.Setenv("ARCHIVE_READ_DAYS", "19")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 7, opts.env.CleanupArchiveReadDays)
}

func TestDefaultCleanupRemoveSessionsDaysValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 30, opts.env.CleanupRemoveSessionsDays)
}

func TestCleanupRemoveSessionsDays(t *testing.T) {
	os.Clearenv()
	t.Setenv("CLEANUP_REMOVE_SESSIONS_DAYS", "7")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 7, opts.env.CleanupRemoveSessionsDays)
}

func TestDefaultWorkerPoolSizeValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.WorkerPoolSize, opts.env.WorkerPoolSize)
}

func TestWorkerPoolSize(t *testing.T) {
	os.Clearenv()
	t.Setenv("WORKER_POOL_SIZE", "42")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 42, opts.env.WorkerPoolSize)
}

func TestDefaultPollingFrequencyValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.PollingFrequency, opts.env.PollingFrequency)
}

func TestPollingFrequency(t *testing.T) {
	os.Clearenv()
	t.Setenv("POLLING_FREQUENCY", "42")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 42, opts.env.PollingFrequency)
}

func TestDefaultForceRefreshInterval(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.ForceRefreshInterval,
		opts.env.ForceRefreshInterval)
}

func TestForceRefreshInterval(t *testing.T) {
	os.Clearenv()
	t.Setenv("FORCE_REFRESH_INTERVAL", "42")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 42, opts.env.ForceRefreshInterval)
}

func TestDefaultBatchSizeValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.BatchSize, opts.env.BatchSize)
}

func TestBatchSize(t *testing.T) {
	os.Clearenv()
	t.Setenv("BATCH_SIZE", "42")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 42, opts.env.BatchSize)
}

func TestDefaultSchedulerRoundRobinValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.SchedulerRoundRobinMinInterval,
		opts.env.SchedulerRoundRobinMinInterval)
}

func TestSchedulerRoundRobin(t *testing.T) {
	os.Clearenv()
	t.Setenv("SCHEDULER_ROUND_ROBIN_MIN_INTERVAL", "15")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 15, opts.env.SchedulerRoundRobinMinInterval)
}

func TestDefaultSchedulerRoundRobinMaxIntervalValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.SchedulerRoundRobinMaxInterval,
		opts.env.SchedulerRoundRobinMaxInterval)
}

func TestSchedulerRoundRobinMaxInterval(t *testing.T) {
	os.Clearenv()
	t.Setenv("SCHEDULER_ROUND_ROBIN_MAX_INTERVAL", "150")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 150, opts.env.SchedulerRoundRobinMaxInterval)
}

func TestPollingParsingErrorLimit(t *testing.T) {
	os.Clearenv()
	t.Setenv("POLLING_PARSING_ERROR_LIMIT", "100")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 100, opts.env.PollingErrorLimit)
}

func TestOAuth2UserCreationWhenUnset(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.False(t, opts.env.Oauth2UserCreationAllowed)
}

func TestOAuth2UserCreationAdmin(t *testing.T) {
	os.Clearenv()
	t.Setenv("OAUTH2_USER_CREATION", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.env.Oauth2UserCreationAllowed)
}

func TestOAuth2ClientID(t *testing.T) {
	os.Clearenv()
	const expected = "foobar"
	t.Setenv("OAUTH2_CLIENT_ID", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.env.Oauth2ClientID)
}

func TestDefaultOAuth2ClientIDValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.Oauth2ClientID, opts.env.Oauth2ClientID)
}

func TestOAuth2ClientSecret(t *testing.T) {
	os.Clearenv()
	const expected = "secret"
	t.Setenv("OAUTH2_CLIENT_SECRET", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.env.Oauth2ClientSecret)
}

func TestDefaultOAuth2ClientSecretValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.Oauth2ClientSecret, opts.env.Oauth2ClientSecret)
}

func TestOAuth2RedirectURL(t *testing.T) {
	os.Clearenv()
	const expected = "http://example.org"
	t.Setenv("OAUTH2_REDIRECT_URL", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.env.Oauth2RedirectURL)
}

func TestDefaultOAuth2RedirectURLValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.Oauth2RedirectURL, opts.env.Oauth2RedirectURL)
}

func TestOAuth2OIDCDiscoveryEndpoint(t *testing.T) {
	os.Clearenv()
	const expected = "http://example.org"
	t.Setenv("OAUTH2_OIDC_DISCOVERY_ENDPOINT", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.env.OidcDiscoveryEndpoint)
}

func TestDefaultOIDCDiscoveryEndpointValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.OidcDiscoveryEndpoint,
		opts.env.OidcDiscoveryEndpoint)
}

func TestOAuth2Provider(t *testing.T) {
	os.Clearenv()
	const expected = "google"
	t.Setenv("OAUTH2_PROVIDER", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.env.Oauth2Provider)
}

func TestDefaultOAuth2ProviderValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.Oauth2Provider, opts.env.Oauth2Provider)
}

func TestHSTSWhenUnset(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.False(t, opts.env.DisableHSTS)
}

func TestHSTS(t *testing.T) {
	os.Clearenv()
	t.Setenv("DISABLE_HSTS", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.env.DisableHSTS)
}

func TestDisableHTTPServiceWhenUnset(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.False(t, opts.env.DisableHttpService)
}

func TestDisableHTTPService(t *testing.T) {
	os.Clearenv()
	t.Setenv("DISABLE_HTTP_SERVICE", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.env.DisableHttpService)
}

func TestDisableSchedulerServiceWhenUnset(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.False(t, opts.env.DisableScheduler)
}

func TestDisableSchedulerService(t *testing.T) {
	os.Clearenv()
	t.Setenv("DISABLE_SCHEDULER_SERVICE", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.env.DisableScheduler)
}

func TestRunMigrationsWhenUnset(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.False(t, opts.env.RunMigrations)
}

func TestRunMigrations(t *testing.T) {
	os.Clearenv()
	t.Setenv("RUN_MIGRATIONS", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.env.RunMigrations)
}

func TestCreateAdminWhenUnset(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.False(t, opts.env.CreateAdmin)
}

func TestCreateAdmin(t *testing.T) {
	os.Clearenv()
	t.Setenv("CREATE_ADMIN", "true")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.env.CreateAdmin)
}

func TestMediaProxyMode(t *testing.T) {
	os.Clearenv()
	const expected = "all"
	t.Setenv("MEDIA_PROXY_MODE", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.env.MediaProxyMode)
}

func TestDefaultMediaProxyModeValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.MediaProxyMode, opts.env.MediaProxyMode)
}

func TestMediaProxyResourceTypes(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image,audio")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, []string{"image", "audio"}, opts.env.MediaProxyResourceTypes)
}

func TestMediaProxyResourceTypesWithDuplicatedValues(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image,audio,image")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, []string{"image", "audio"}, opts.env.MediaProxyResourceTypes)
}

func TestDefaultMediaProxyResourceTypes(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, []string{"image"}, opts.env.MediaProxyResourceTypes)
}

func TestMediaProxyHTTPClientTimeout(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_HTTP_CLIENT_TIMEOUT", "24")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 24, opts.env.MediaProxyHTTPClientTimeout)
}

func TestDefaultMediaProxyHTTPClientTimeoutValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.MediaProxyHTTPClientTimeout,
		opts.env.MediaProxyHTTPClientTimeout)
}

func TestMediaProxyCustomURL(t *testing.T) {
	os.Clearenv()
	const expected = "http://example.org/proxy"
	t.Setenv("MEDIA_PROXY_CUSTOM_URL", expected)
	opts := parseEnvironmentVariables(t)
	require.NotNil(t, opts.env.MediaProxyCustomURL)
	assert.Equal(t, expected, opts.env.MediaProxyCustomURL.String())
}

func TestMediaProxyPrivateKey(t *testing.T) {
	os.Clearenv()
	const foobar = "foobar"
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", foobar)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, foobar, opts.env.MediaProxyPrivateKey)
}

func TestHTTPSOff(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.False(t, opts.env.HTTPS)
}

func TestHTTPSOn(t *testing.T) {
	os.Clearenv()
	t.Setenv("HTTPS", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.env.HTTPS)
}

func TestHTTPClientTimeout(t *testing.T) {
	os.Clearenv()
	t.Setenv("HTTP_CLIENT_TIMEOUT", "42")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 42, opts.env.HttpClientTimeout)
}

func TestDefaultHTTPClientTimeoutValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.HttpClientTimeout, opts.env.HttpClientTimeout)
}

func TestHTTPClientMaxBodySize(t *testing.T) {
	os.Clearenv()
	t.Setenv("HTTP_CLIENT_MAX_BODY_SIZE", "42")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, int64(42*1024*1024), opts.env.HttpClientMaxBodySize)
}

func TestDefaultHTTPClientMaxBodySizeValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.HttpClientMaxBodySize*1024*1024,
		opts.env.HttpClientMaxBodySize)
}

func TestHTTPServerTimeout(t *testing.T) {
	os.Clearenv()
	t.Setenv("HTTP_SERVER_TIMEOUT", "342")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 342, opts.env.HttpServerTimeout)
}

func TestDefaultHTTPServerTimeoutValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.HttpServerTimeout, opts.env.HttpServerTimeout)
}

func TestParseConfigFile(t *testing.T) {
	content := []byte(`
 # This is a comment

LOG_LEVEL = debug
`)

	tmpfile, err := os.CreateTemp(t.TempDir(), "miniflux.*.unit_test.conf")
	require.NoError(t, err)
	_, err = tmpfile.Write(content)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	os.Clearenv()
	parser := NewParser()
	opts, err := parser.ParseEnvFile(tmpfile.Name())
	require.NoError(t, err)
	require.NotNil(t, opts)

	assert.Equal(t, "debug", opts.env.LogLevel)
}

func TestParseConfigFile_invalid(t *testing.T) {
	invalidContent := []byte(`
 # This is a comment

LOG_LEVEL = debug

Invalid text
`)

	tmpfile, err := os.CreateTemp(t.TempDir(), "miniflux.*.unit_test.conf")
	require.NoError(t, err)
	_, err = tmpfile.Write(invalidContent)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	os.Clearenv()
	parser := NewParser()
	_, err = parser.ParseEnvFile(tmpfile.Name())
	t.Log(err)
	require.ErrorContains(t, err, "Invalid text")
}

func TestAuthProxyHeader(t *testing.T) {
	os.Clearenv()
	const expected = "X-Forwarded-User"
	t.Setenv("AUTH_PROXY_HEADER", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.env.AuthProxyHeader)
}

func TestDefaultAuthProxyHeaderValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().env.AuthProxyHeader, opts.env.AuthProxyHeader)
}

func TestAuthProxyUserCreationWhenUnset(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.False(t, opts.env.AuthProxyUserCreation)
}

func TestAuthProxyUserCreationAdmin(t *testing.T) {
	os.Clearenv()
	t.Setenv("AUTH_PROXY_USER_CREATION", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.env.AuthProxyUserCreation)
}

func TestFetchBilibiliWatchTime(t *testing.T) {
	os.Clearenv()
	t.Setenv("FETCH_BILIBILI_WATCH_TIME", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.env.FetchBilibiliWatchTime)
}

func TestFetchNebulaWatchTime(t *testing.T) {
	os.Clearenv()
	t.Setenv("FETCH_NEBULA_WATCH_TIME", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.env.FetchNebulaWatchTime)
}

func TestFetchOdyseeWatchTime(t *testing.T) {
	os.Clearenv()
	t.Setenv("FETCH_ODYSEE_WATCH_TIME", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.env.FetchOdyseeWatchTime)
}

func TestFetchYouTubeWatchTime(t *testing.T) {
	os.Clearenv()
	t.Setenv("FETCH_YOUTUBE_WATCH_TIME", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.env.FetchYouTubeWatchTime)
}

func TestYouTubeApiKey(t *testing.T) {
	os.Clearenv()
	const expected = "AAAAAAAAAAAAAaaaaaaaaaaaaa0000000000000"
	t.Setenv("YOUTUBE_API_KEY", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.env.YouTubeApiKey)
}

func TestDefaultYouTubeEmbedUrl(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	const expected = "https://www.youtube-nocookie.com/embed/"
	assert.Equal(t, expected, opts.env.YouTubeEmbedUrlOverride.String())
	assert.Equal(t, "www.youtube-nocookie.com", opts.env.YouTubeEmbedUrlOverride.Hostname())
}

func TestYouTubeEmbedUrlOverride(t *testing.T) {
	os.Clearenv()
	const expected = "https://invidious.custom/embed/"
	t.Setenv("YOUTUBE_EMBED_URL_OVERRIDE", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.env.YouTubeEmbedUrlOverride.String())
	assert.Equal(t, "invidious.custom", opts.env.YouTubeEmbedUrlOverride.Hostname())
}

func TestParseConfigDumpOutput(t *testing.T) {
	wantOpts := parseEnvironmentVariables(t)
	wantOpts.env.AdminUsername = "my-username"

	serialized := wantOpts.String()
	tmpfile, err := os.CreateTemp(t.TempDir(), "miniflux.*.unit_test.conf")
	require.NoError(t, err)

	_, err = tmpfile.WriteString(serialized)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	os.Clearenv()
	parsedOpts, err := NewParser().ParseEnvFile(tmpfile.Name())
	require.NoError(t, err)
	require.NotNil(t, parsedOpts)
	assert.Equal(t, wantOpts.env.AdminUsername, parsedOpts.env.AdminUsername)
}

func TestHTTPClientProxies(t *testing.T) {
	os.Clearenv()
	const proxy1 = "http://proxy1.example.com"
	const proxy2 = "http://proxy2.example.com"
	t.Setenv("HTTP_CLIENT_PROXIES", proxy1+","+proxy2)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, []string{proxy1, proxy2}, opts.env.HttpClientProxies)
}

func TestDefaultHTTPClientProxiesValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Empty(t, opts.env.HttpClientProxies)
}

func TestHTTPClientProxy(t *testing.T) {
	os.Clearenv()
	const expected = "http://proxy.example.com"
	t.Setenv("HTTP_CLIENT_PROXY", expected)
	opts := parseEnvironmentVariables(t)
	require.NotNil(t, opts.env.HttpClientProxyURL)
	assert.Equal(t, expected, opts.env.HttpClientProxyURL.String())
}

func TestInvalidHTTPClientProxy(t *testing.T) {
	os.Clearenv()
	t.Setenv("HTTP_CLIENT_PROXY", "sche|me://invalid-proxy-url")
	_, err := NewParser().ParseEnvironmentVariables()
	t.Log(err)
	require.ErrorContains(t, err, "HttpClientProxyURL")
}

func TestOptions_Logging_default(t *testing.T) {
	os.Clearenv()
	opts = parseEnvironmentVariables(t)
	want := []Log{
		{
			LogFile:     LogFile(),
			LogDateTime: LogDateTime(),
			LogFormat:   LogFormat(),
			LogLevel:    LogLevel(),
		},
	}
	assert.Equal(t, want, Logging())
}

func TestOptions_Logging_multiple(t *testing.T) {
	os.Clearenv()
	t.Setenv("LOG_0_FILE", "stderr")
	t.Setenv("LOG_0_FORMAT", "human")
	t.Setenv("LOG_0_LEVEL", "warning")
	t.Setenv("LOG_1_FILE", "/var/log/miniflux/miniflux.log")
	t.Setenv("LOG_1_DATE_TIME", "true")
	t.Setenv("LOG_1_FORMAT", "human")
	t.Setenv("LOG_1_LEVEL", "info")
	opts = parseEnvironmentVariables(t)
	want := []Log{
		{
			LogFile:   "stderr",
			LogFormat: "human",
			LogLevel:  "warning",
		},
		{
			LogFile:     "/var/log/miniflux/miniflux.log",
			LogDateTime: true,
			LogFormat:   "human",
			LogLevel:    "info",
		},
	}
	assert.Equal(t, want, Logging())
}

func TestOptions_Logging_priority(t *testing.T) {
	os.Clearenv()
	t.Setenv("LOG_FILE", "stdout")
	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_0_FILE", "stderr")
	t.Setenv("LOG_0_FORMAT", "human")
	t.Setenv("LOG_0_LEVEL", "warning")
	opts = parseEnvironmentVariables(t)
	want := []Log{
		{
			LogFile:   "stderr",
			LogFormat: "human",
			LogLevel:  "warning",
		},
	}
	assert.Equal(t, want, Logging())
}

func TestTrustedProxies(t *testing.T) {
	opts := parseEnvironmentVariables(t)
	assert.Contains(t, opts.env.TrustedProxies, "127.0.0.1")
	assert.NotContains(t, opts.env.TrustedProxies, "127.0.0.2")

	os.Clearenv()
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1")
	opts = parseEnvironmentVariables(t)
	assert.Contains(t, opts.env.TrustedProxies, "127.0.0.1")

	t.Setenv("TRUSTED_PROXIES", "127.0.0.1,127.0.0.2")
	opts = parseEnvironmentVariables(t)
	assert.Contains(t, opts.env.TrustedProxies, "127.0.0.1")
	assert.Contains(t, opts.trustedProxies, "127.0.0.2")
}

func TestFetcherAllowPrivateNetworks(t *testing.T) {
	assert.False(t, FetcherAllowPrivateNetworks())

	os.Clearenv()
	t.Setenv("FETCHER_ALLOW_PRIVATE_NETWORKS", "1")
	require.NoError(t, Load(""))
	assert.True(t, FetcherAllowPrivateNetworks())

	t.Setenv("FETCHER_ALLOW_PRIVATE_NETWORKS", "0")
	require.NoError(t, Load(""))
	assert.False(t, FetcherAllowPrivateNetworks())
}

func TestLoadYAML(t *testing.T) {
	require.Error(t, LoadYAML("testdata/notfound.yaml", ""))
	require.Error(t, LoadYAML("", "testdata/notfound.env"))

	os.Clearenv()
	require.NoError(t, LoadYAML("", ""))
	require.NoError(t, LoadYAML("testdata/host_limits.yaml", ""))

	require.NoError(t, LoadYAML("testdata/host_limits.yaml",
		"testdata/host_limits.env"))
	assert.Equal(t, int64(4), ConnectionsPerServer())
	//nolint:testifylint // const 1 is always 1.0
	assert.Equal(t, float64(1), RateLimitPerServer())

	require.NoError(t, LoadYAML("", "testdata/host_limits.env"))
	assert.Equal(t, int64(4), ConnectionsPerServer())
	//nolint:testifylint // const 1 is always 1.0
	assert.Equal(t, float64(1), RateLimitPerServer())
}

func TestLoadYAML_findHostLimits(t *testing.T) {
	os.Clearenv()
	require.NoError(t, LoadYAML("testdata/host_limits.yaml", ""))

	tests := []struct {
		name string
		want HostLimits
	}{
		{
			name: "default",
			want: HostLimits{
				Connections: ConnectionsPerServer(),
				Rate:        RateLimitPerServer(),
			},
		},
		{
			name: "localhost",
			want: HostLimits{Connections: 3, Rate: 100},
		},
		{
			name: "a.example.com",
			want: HostLimits{Connections: ConnectionsPerServer(), Rate: 15},
		},
		{
			name: "b.example.com",
			want: HostLimits{Connections: 5, Rate: RateLimitPerServer()},
		},
		{
			name: "c.example.com",
			want: HostLimits{Connections: ConnectionsPerServer(), Rate: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FindHostLimits(tt.name))
		})
	}
}
