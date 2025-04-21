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
	assert.Equal(t, NewOptions().LogFile(), opts.LogFile())
}

func parseEnvironmentVariables(t *testing.T) *Options {
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
	assert.Equal(t, want, opts.LogFile())
}

func TestLogFileWithEmptyValue(t *testing.T) {
	os.Clearenv()
	t.Setenv("LOG_FILE", "")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().LogFile(), opts.LogFile())
}

func TestLogLevelDefaultValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().LogLevel(), opts.LogLevel())
}

func TestLogLevelWithCustomValue(t *testing.T) {
	os.Clearenv()
	const want = "warning"
	t.Setenv("LOG_LEVEL", want)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, want, opts.LogLevel())
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
	assert.Equal(t, NewOptions().LogDateTime(), opts.LogDateTime())
}

func TestLogDateTimeWithCustomValue(t *testing.T) {
	os.Clearenv()
	t.Setenv("LOG_DATE_TIME", "true")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.LogDateTime())
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
	assert.Equal(t, NewOptions().LogFormat(), opts.LogFormat())
}

func TestLogFormatWithCustomValue(t *testing.T) {
	os.Clearenv()
	const want = "json"
	t.Setenv("LOG_FORMAT", want)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, want, opts.LogFormat())
}

func TestLogFormatWithInvalidValue(t *testing.T) {
	os.Clearenv()
	t.Setenv("LOG_FORMAT", "invalid")
	_, err := NewParser().ParseEnvironmentVariables()
	t.Log(err)
	require.ErrorContains(t, err, "failed on the 'oneof' tag")
}

func TestDebugModeOn(t *testing.T) {
	os.Clearenv()
	t.Setenv("DEBUG", "1")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, "debug", opts.LogLevel())
}

func TestDebugModeOff(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, "info", opts.LogLevel())
}

func TestCustomBaseURL(t *testing.T) {
	os.Clearenv()
	const want = "http://example.org"
	t.Setenv("BASE_URL", want)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, want, opts.BaseURL())
	assert.Equal(t, want, opts.RootURL())
	assert.Empty(t, opts.BasePath())
}

func TestCustomBaseURLWithTrailingSlash(t *testing.T) {
	os.Clearenv()
	t.Setenv("BASE_URL", "http://example.org/folder/")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, "http://example.org/folder", opts.BaseURL())
	assert.Equal(t, "http://example.org", opts.RootURL())
	assert.Equal(t, "/folder", opts.BasePath())
}

func TestCustomBaseURLWithCustomPort(t *testing.T) {
	os.Clearenv()
	t.Setenv("BASE_URL", "http://example.org:88/folder/")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, "http://example.org:88/folder", opts.BaseURL())
	assert.Equal(t, "http://example.org:88", opts.RootURL())
	assert.Equal(t, "/folder", opts.BasePath())
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
	assert.Equal(t, defs.BaseURL(), opts.BaseURL())
	assert.Equal(t, defs.RootURL(), opts.RootURL())
	assert.Empty(t, opts.BasePath())
}

func TestDatabaseURL(t *testing.T) {
	os.Clearenv()
	const expected = "foobar"
	t.Setenv("DATABASE_URL", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.DatabaseURL())
	assert.False(t, opts.IsDefaultDatabaseURL())
}

func TestDefaultDatabaseURLValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().DatabaseURL(), opts.DatabaseURL())
	assert.True(t, opts.IsDefaultDatabaseURL())
}

func TestDefaultDatabaseMaxConnsValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().DatabaseMaxConns(), opts.DatabaseMaxConns())
}

func TestDatabaseMaxConns(t *testing.T) {
	os.Clearenv()
	t.Setenv("DATABASE_MAX_CONNS", "42")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 42, opts.DatabaseMaxConns())
}

func TestDefaultDatabaseMinConnsValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().DatabaseMinConns(), opts.DatabaseMinConns())
}

func TestDatabaseMinConns(t *testing.T) {
	os.Clearenv()
	t.Setenv("DATABASE_MIN_CONNS", "42")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 42, opts.DatabaseMinConns())
}

func TestListenAddr(t *testing.T) {
	os.Clearenv()
	const expected = "foobar"
	t.Setenv("LISTEN_ADDR", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.ListenAddr())
}

func TestListenAddrWithPortDefined(t *testing.T) {
	os.Clearenv()
	t.Setenv("PORT", "3000")
	t.Setenv("LISTEN_ADDR", "foobar")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, ":3000", opts.ListenAddr())
}

func TestDefaultListenAddrValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().ListenAddr(), opts.ListenAddr())
}

func TestCertFile(t *testing.T) {
	os.Clearenv()
	const expected = "foobar"
	t.Setenv("CERT_FILE", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.CertFile())
}

func TestDefaultCertFileValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().CertFile(), opts.CertFile())
}

func TestKeyFile(t *testing.T) {
	os.Clearenv()
	const expected = "foobar"
	t.Setenv("KEY_FILE", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.CertKeyFile())
}

func TestDefaultKeyFileValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().CertKeyFile(), opts.CertKeyFile())
}

func TestCertDomain(t *testing.T) {
	os.Clearenv()
	const expected = "example.org"
	t.Setenv("CERT_DOMAIN", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.CertDomain())
}

func TestDefaultCertDomainValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().CertDomain(), opts.CertDomain())
}

func TestDefaultCleanupFrequencyHoursValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().CleanupFrequencyHours(),
		opts.CleanupFrequencyHours())
}

func TestCleanupFrequencyHours(t *testing.T) {
	os.Clearenv()
	t.Setenv("CLEANUP_FREQUENCY_HOURS", "42")
	t.Setenv("CLEANUP_FREQUENCY", "19")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 42, opts.CleanupFrequencyHours())
}

func TestDefaultCleanupArchiveReadDaysValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 60, opts.CleanupArchiveReadDays())
}

func TestCleanupArchiveReadDays(t *testing.T) {
	os.Clearenv()
	t.Setenv("CLEANUP_ARCHIVE_READ_DAYS", "7")
	t.Setenv("ARCHIVE_READ_DAYS", "19")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 7, opts.CleanupArchiveReadDays())
}

func TestDefaultCleanupRemoveSessionsDaysValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 30, opts.CleanupRemoveSessionsDays())
}

func TestCleanupRemoveSessionsDays(t *testing.T) {
	os.Clearenv()
	t.Setenv("CLEANUP_REMOVE_SESSIONS_DAYS", "7")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 7, opts.CleanupRemoveSessionsDays())
}

func TestDefaultWorkerPoolSizeValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().WorkerPoolSize(), opts.WorkerPoolSize())
}

func TestWorkerPoolSize(t *testing.T) {
	os.Clearenv()
	t.Setenv("WORKER_POOL_SIZE", "42")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 42, opts.WorkerPoolSize())
}

func TestDefautPollingFrequencyValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().PollingFrequency(), opts.PollingFrequency())
}

func TestPollingFrequency(t *testing.T) {
	os.Clearenv()
	t.Setenv("POLLING_FREQUENCY", "42")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 42, opts.PollingFrequency())
}

func TestDefautForceRefreshInterval(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().ForceRefreshInterval(),
		opts.ForceRefreshInterval())
}

func TestForceRefreshInterval(t *testing.T) {
	os.Clearenv()
	t.Setenv("FORCE_REFRESH_INTERVAL", "42")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 42, opts.ForceRefreshInterval())
}

func TestDefaultBatchSizeValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().BatchSize(), opts.BatchSize())
}

func TestBatchSize(t *testing.T) {
	os.Clearenv()
	t.Setenv("BATCH_SIZE", "42")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 42, opts.BatchSize())
}

func TestDefautPollingSchedulerValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().PollingScheduler(), opts.PollingScheduler())
}

func TestPollingScheduler(t *testing.T) {
	os.Clearenv()
	const expected = "entry_frequency"
	t.Setenv("POLLING_SCHEDULER", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.PollingScheduler())
}

func TestDefautSchedulerEntryFrequencyMaxIntervalValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().SchedulerEntryFrequencyMaxInterval(),
		opts.SchedulerEntryFrequencyMaxInterval())
}

func TestSchedulerEntryFrequencyMaxInterval(t *testing.T) {
	os.Clearenv()
	t.Setenv("SCHEDULER_ENTRY_FREQUENCY_MAX_INTERVAL", "30")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 30, opts.SchedulerEntryFrequencyMaxInterval())
}

func TestDefautSchedulerEntryFrequencyMinIntervalValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().SchedulerEntryFrequencyMinInterval(),
		opts.SchedulerEntryFrequencyMinInterval())
}

func TestSchedulerEntryFrequencyMinInterval(t *testing.T) {
	os.Clearenv()
	t.Setenv("SCHEDULER_ENTRY_FREQUENCY_MIN_INTERVAL", "30")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 30, opts.SchedulerEntryFrequencyMinInterval())
}

func TestDefautSchedulerEntryFrequencyFactorValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().SchedulerEntryFrequencyFactor(),
		opts.SchedulerEntryFrequencyFactor())
}

func TestSchedulerEntryFrequencyFactor(t *testing.T) {
	os.Clearenv()
	t.Setenv("SCHEDULER_ENTRY_FREQUENCY_FACTOR", "2")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 2, opts.SchedulerEntryFrequencyFactor())
}

func TestDefaultSchedulerRoundRobinValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().SchedulerRoundRobinMinInterval(),
		opts.SchedulerRoundRobinMinInterval())
}

func TestSchedulerRoundRobin(t *testing.T) {
	os.Clearenv()
	t.Setenv("SCHEDULER_ROUND_ROBIN_MIN_INTERVAL", "15")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 15, opts.SchedulerRoundRobinMinInterval())
}

func TestDefaultSchedulerRoundRobinMaxIntervalValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().SchedulerRoundRobinMaxInterval(),
		opts.SchedulerRoundRobinMaxInterval())
}

func TestSchedulerRoundRobinMaxInterval(t *testing.T) {
	os.Clearenv()
	t.Setenv("SCHEDULER_ROUND_ROBIN_MAX_INTERVAL", "150")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 150, opts.SchedulerRoundRobinMaxInterval())
}

func TestPollingParsingErrorLimit(t *testing.T) {
	os.Clearenv()
	t.Setenv("POLLING_PARSING_ERROR_LIMIT", "100")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 100, opts.PollingParsingErrorLimit())
}

func TestOAuth2UserCreationWhenUnset(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.False(t, opts.IsOAuth2UserCreationAllowed())
}

func TestOAuth2UserCreationAdmin(t *testing.T) {
	os.Clearenv()
	t.Setenv("OAUTH2_USER_CREATION", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.IsOAuth2UserCreationAllowed())
}

func TestOAuth2ClientID(t *testing.T) {
	os.Clearenv()
	const expected = "foobar"
	t.Setenv("OAUTH2_CLIENT_ID", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.OAuth2ClientID())
}

func TestDefaultOAuth2ClientIDValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().OAuth2ClientID(), opts.OAuth2ClientID())
}

func TestOAuth2ClientSecret(t *testing.T) {
	os.Clearenv()
	const expected = "secret"
	t.Setenv("OAUTH2_CLIENT_SECRET", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.OAuth2ClientSecret())
}

func TestDefaultOAuth2ClientSecretValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().OAuth2ClientSecret(), opts.OAuth2ClientSecret())
}

func TestOAuth2RedirectURL(t *testing.T) {
	os.Clearenv()
	const expected = "http://example.org"
	t.Setenv("OAUTH2_REDIRECT_URL", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.OAuth2RedirectURL())
}

func TestDefaultOAuth2RedirectURLValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().OAuth2RedirectURL(), opts.OAuth2RedirectURL())
}

func TestOAuth2OIDCDiscoveryEndpoint(t *testing.T) {
	os.Clearenv()
	const expected = "http://example.org"
	t.Setenv("OAUTH2_OIDC_DISCOVERY_ENDPOINT", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.OIDCDiscoveryEndpoint())
}

func TestDefaultOIDCDiscoveryEndpointValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().OIDCDiscoveryEndpoint(),
		opts.OIDCDiscoveryEndpoint())
}

func TestOAuth2Provider(t *testing.T) {
	os.Clearenv()
	const expected = "google"
	t.Setenv("OAUTH2_PROVIDER", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.OAuth2Provider())
}

func TestDefaultOAuth2ProviderValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().OAuth2Provider(), opts.OAuth2Provider())
}

func TestHSTSWhenUnset(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.HasHSTS())
}

func TestHSTS(t *testing.T) {
	os.Clearenv()
	t.Setenv("DISABLE_HSTS", "1")
	opts := parseEnvironmentVariables(t)
	assert.False(t, opts.HasHSTS())
}

func TestDisableHTTPServiceWhenUnset(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.HasHTTPService())
}

func TestDisableHTTPService(t *testing.T) {
	os.Clearenv()
	t.Setenv("DISABLE_HTTP_SERVICE", "1")
	opts := parseEnvironmentVariables(t)
	assert.False(t, opts.HasHTTPService())
}

func TestDisableSchedulerServiceWhenUnset(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.HasSchedulerService())
}

func TestDisableSchedulerService(t *testing.T) {
	os.Clearenv()
	t.Setenv("DISABLE_SCHEDULER_SERVICE", "1")
	opts := parseEnvironmentVariables(t)
	assert.False(t, opts.HasSchedulerService())
}

func TestRunMigrationsWhenUnset(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.False(t, opts.RunMigrations())
}

func TestRunMigrations(t *testing.T) {
	os.Clearenv()
	t.Setenv("RUN_MIGRATIONS", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.RunMigrations())
}

func TestCreateAdminWhenUnset(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.False(t, opts.CreateAdmin())
}

func TestCreateAdmin(t *testing.T) {
	os.Clearenv()
	t.Setenv("CREATE_ADMIN", "true")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.CreateAdmin())
}

func TestPocketConsumerKeyFromEnvVariable(t *testing.T) {
	os.Clearenv()
	const expected = "something"
	t.Setenv("POCKET_CONSUMER_KEY", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.PocketConsumerKey("default"))
}

func TestPocketConsumerKeyFromUserPrefs(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	const expected = "default"
	assert.Equal(t, expected, opts.PocketConsumerKey(expected))
}

func TestMediaProxyMode(t *testing.T) {
	os.Clearenv()
	const expected = "all"
	t.Setenv("MEDIA_PROXY_MODE", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.MediaProxyMode())
}

func TestDefaultMediaProxyModeValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().MediaProxyMode(), opts.MediaProxyMode())
}

func TestMediaProxyResourceTypes(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image,audio")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, []string{"image", "audio"}, opts.MediaProxyResourceTypes())
}

func TestMediaProxyResourceTypesWithDuplicatedValues(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image,audio, image")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, []string{"image", "audio"}, opts.MediaProxyResourceTypes())
}

func TestDefaultMediaProxyResourceTypes(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, []string{"image"}, opts.MediaProxyResourceTypes())
}

func TestMediaProxyHTTPClientTimeout(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_HTTP_CLIENT_TIMEOUT", "24")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 24, opts.MediaProxyHTTPClientTimeout())
}

func TestDefaultMediaProxyHTTPClientTimeoutValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().MediaProxyHTTPClientTimeout(),
		opts.MediaProxyHTTPClientTimeout())
}

func TestMediaProxyCustomURL(t *testing.T) {
	os.Clearenv()
	const expected = "http://example.org/proxy"
	t.Setenv("MEDIA_PROXY_CUSTOM_URL", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.MediaCustomProxyURL())
}

func TestMediaProxyPrivateKey(t *testing.T) {
	os.Clearenv()
	const foobar = "foobar"
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", foobar)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, []byte(foobar), opts.MediaProxyPrivateKey())
}

func TestProxyImagesOptionForBackwardCompatibility(t *testing.T) {
	os.Clearenv()
	const expectedProxyOption = "all"
	t.Setenv("PROXY_IMAGES", expectedProxyOption)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, []string{"image"}, opts.MediaProxyResourceTypes())
	assert.Equal(t, expectedProxyOption, opts.MediaProxyMode())
}

func TestProxyImageURLForBackwardCompatibility(t *testing.T) {
	os.Clearenv()
	const expected = "http://example.org/proxy"
	t.Setenv("PROXY_IMAGE_URL", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.MediaCustomProxyURL())
}

func TestProxyURLOptionForBackwardCompatibility(t *testing.T) {
	os.Clearenv()
	const expected = "http://example.org/proxy"
	t.Setenv("PROXY_URL", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.MediaCustomProxyURL())
}

func TestProxyMediaTypesOptionForBackwardCompatibility(t *testing.T) {
	os.Clearenv()
	t.Setenv("PROXY_MEDIA_TYPES", "image,audio")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, []string{"image", "audio"}, opts.MediaProxyResourceTypes())
}

func TestProxyOptionForBackwardCompatibility(t *testing.T) {
	os.Clearenv()
	const expected = "all"
	t.Setenv("PROXY_OPTION", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.MediaProxyMode())
}

func TestProxyHTTPClientTimeoutOptionForBackwardCompatibility(t *testing.T) {
	os.Clearenv()
	t.Setenv("PROXY_HTTP_CLIENT_TIMEOUT", "24")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 24, opts.MediaProxyHTTPClientTimeout())
}

func TestProxyPrivateKeyOptionForBackwardCompatibility(t *testing.T) {
	os.Clearenv()
	const foobar = "foobar"
	t.Setenv("PROXY_PRIVATE_KEY", foobar)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, []byte(foobar), opts.MediaProxyPrivateKey())
}

func TestHTTPSOff(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.False(t, opts.HTTPS())
}

func TestHTTPSOn(t *testing.T) {
	os.Clearenv()
	t.Setenv("HTTPS", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.HTTPS())
}

func TestHTTPClientTimeout(t *testing.T) {
	os.Clearenv()
	t.Setenv("HTTP_CLIENT_TIMEOUT", "42")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 42, opts.HTTPClientTimeout())
}

func TestDefaultHTTPClientTimeoutValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().HTTPClientTimeout(), opts.HTTPClientTimeout())
}

func TestHTTPClientMaxBodySize(t *testing.T) {
	os.Clearenv()
	t.Setenv("HTTP_CLIENT_MAX_BODY_SIZE", "42")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, int64(42*1024*1024), opts.HTTPClientMaxBodySize())
}

func TestDefaultHTTPClientMaxBodySizeValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().HTTPClientMaxBodySize()*1024*1024,
		opts.HTTPClientMaxBodySize())
}

func TestHTTPServerTimeout(t *testing.T) {
	os.Clearenv()
	t.Setenv("HTTP_SERVER_TIMEOUT", "342")
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, 342, opts.HTTPServerTimeout())
}

func TestDefaultHTTPServerTimeoutValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().HTTPServerTimeout(), opts.HTTPServerTimeout())
}

func TestParseConfigFile(t *testing.T) {
	content := []byte(`
 # This is a comment

DEBUG = 1

 POCKET_CONSUMER_KEY= >#1234
`)

	tmpfile, err := os.CreateTemp(t.TempDir(), "miniflux.*.unit_test.conf")
	require.NoError(t, err)
	_, err = tmpfile.Write(content)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	os.Clearenv()
	parser := NewParser()
	opts, err := parser.ParseFile(tmpfile.Name())
	require.NoError(t, err)
	require.NotNil(t, opts)

	assert.Equal(t, "debug", opts.LogLevel())
	assert.Equal(t, ">#1234", opts.PocketConsumerKey("default"))
}

func TestParseConfigFile_invalid(t *testing.T) {
	invalidContent := []byte(`
 # This is a comment

DEBUG = 1

 POCKET_CONSUMER_KEY= >#1234

Invalid text
`)

	tmpfile, err := os.CreateTemp(t.TempDir(), "miniflux.*.unit_test.conf")
	require.NoError(t, err)
	_, err = tmpfile.Write(invalidContent)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	os.Clearenv()
	parser := NewParser()
	_, err = parser.ParseFile(tmpfile.Name())
	t.Log(err)
	require.ErrorContains(t, err, "Invalid text")
}

func TestAuthProxyHeader(t *testing.T) {
	os.Clearenv()
	const expected = "X-Forwarded-User"
	t.Setenv("AUTH_PROXY_HEADER", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.AuthProxyHeader())
}

func TestDefaultAuthProxyHeaderValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, NewOptions().AuthProxyHeader(), opts.AuthProxyHeader())
}

func TestAuthProxyUserCreationWhenUnset(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.False(t, opts.IsAuthProxyUserCreationAllowed())
}

func TestAuthProxyUserCreationAdmin(t *testing.T) {
	os.Clearenv()
	t.Setenv("AUTH_PROXY_USER_CREATION", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.IsAuthProxyUserCreationAllowed())
}

func TestFetchBilibiliWatchTime(t *testing.T) {
	os.Clearenv()
	t.Setenv("FETCH_BILIBILI_WATCH_TIME", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.FetchBilibiliWatchTime())
}

func TestFetchNebulaWatchTime(t *testing.T) {
	os.Clearenv()
	t.Setenv("FETCH_NEBULA_WATCH_TIME", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.FetchNebulaWatchTime())
}

func TestFetchOdyseeWatchTime(t *testing.T) {
	os.Clearenv()
	t.Setenv("FETCH_ODYSEE_WATCH_TIME", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.FetchOdyseeWatchTime())
}

func TestFetchYouTubeWatchTime(t *testing.T) {
	os.Clearenv()
	t.Setenv("FETCH_YOUTUBE_WATCH_TIME", "1")
	opts := parseEnvironmentVariables(t)
	assert.True(t, opts.FetchYouTubeWatchTime())
}

func TestYouTubeApiKey(t *testing.T) {
	os.Clearenv()
	const expected = "AAAAAAAAAAAAAaaaaaaaaaaaaa0000000000000"
	t.Setenv("YOUTUBE_API_KEY", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.YouTubeApiKey())
}

func TestYouTubeEmbedUrlOverride(t *testing.T) {
	os.Clearenv()
	const expected = "https://invidious.custom/embed/"
	t.Setenv("YOUTUBE_EMBED_URL_OVERRIDE", expected)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, expected, opts.YouTubeEmbedUrlOverride())
}

func TestParseConfigDumpOutput(t *testing.T) {
	wantOpts := NewOptions()
	wantOpts.env.AdminUsername = "my-username"

	serialized := wantOpts.String()
	tmpfile, err := os.CreateTemp(t.TempDir(), "miniflux.*.unit_test.conf")
	require.NoError(t, err)

	_, err = tmpfile.WriteString(serialized)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	os.Clearenv()
	parsedOpts, err := NewParser().ParseFile(tmpfile.Name())
	require.NoError(t, err)
	require.NotNil(t, parsedOpts)
	assert.Equal(t, wantOpts.AdminUsername(), parsedOpts.AdminUsername())
}

func TestHTTPClientProxies(t *testing.T) {
	os.Clearenv()
	const proxy1 = "http://proxy1.example.com"
	const proxy2 = "http://proxy2.example.com"
	t.Setenv("HTTP_CLIENT_PROXIES", proxy1+","+proxy2)
	opts := parseEnvironmentVariables(t)
	assert.Equal(t, []string{proxy1, proxy2}, opts.HTTPClientProxies())
}

func TestDefaultHTTPClientProxiesValue(t *testing.T) {
	os.Clearenv()
	opts := parseEnvironmentVariables(t)
	assert.Empty(t, opts.HTTPClientProxies())
}

func TestHTTPClientProxy(t *testing.T) {
	os.Clearenv()
	const expected = "http://proxy.example.com"
	t.Setenv("HTTP_CLIENT_PROXY", expected)
	opts := parseEnvironmentVariables(t)
	require.NotNil(t, opts.HTTPClientProxyURL())
	assert.Equal(t, expected, opts.HTTPClientProxyURL().String())
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
	opts := parseEnvironmentVariables(t)
	want := []Log{
		{
			LogFile:     opts.LogFile(),
			LogDateTime: opts.LogDateTime(),
			LogFormat:   opts.LogFormat(),
			LogLevel:    opts.LogLevel(),
		},
	}
	assert.Equal(t, want, opts.Logging())
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
	opts := parseEnvironmentVariables(t)
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
	assert.Equal(t, want, opts.Logging())
}

func TestOptions_Logging_priority(t *testing.T) {
	os.Clearenv()
	t.Setenv("LOG_FILE", "stdout")
	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_0_FILE", "stderr")
	t.Setenv("LOG_0_FORMAT", "human")
	t.Setenv("LOG_0_LEVEL", "warning")
	opts := parseEnvironmentVariables(t)
	want := []Log{
		{
			LogFile:   "stderr",
			LogFormat: "human",
			LogLevel:  "warning",
		},
	}
	assert.Equal(t, want, opts.Logging())
}
