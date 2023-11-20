// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public Licensee as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public Licensee for more details.
//
// You should have received a copy of the GNU Affero General Public Licensee
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package config

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	_ "time/tzdata" // Imports time zone data instead of relying on the os

	"code.vikunja.io/api/pkg/log"
	"github.com/spf13/viper"
)

// Key is used as a config key
type Key string

// These constants hold all config value keys
const (
	// #nosec
	ServiceJWTSecret       Key = `service.JWTSecret`
	ServiceJWTTTL          Key = `service.jwtttl`
	ServiceJWTTTLLong      Key = `service.jwtttllong`
	ServiceInterface       Key = `service.interface`
	ServiceUnixSocket      Key = `service.unixsocket`
	ServiceUnixSocketMode  Key = `service.unixsocketmode`
	ServiceFrontendurl     Key = `service.frontendurl`
	ServiceEnableCaldav    Key = `service.enablecaldav`
	ServiceRootpath        Key = `service.rootpath`
	ServiceStaticpath      Key = `service.staticpath`
	ServiceMaxItemsPerPage Key = `service.maxitemsperpage`
	ServiceDemoMode        Key = `service.demomode`
	// Deprecated: Use metrics.enabled
	ServiceEnableMetrics         Key = `service.enablemetrics`
	ServiceMotd                  Key = `service.motd`
	ServiceEnableLinkSharing     Key = `service.enablelinksharing`
	ServiceEnableRegistration    Key = `service.enableregistration`
	ServiceEnableTaskAttachments Key = `service.enabletaskattachments`
	ServiceTimeZone              Key = `service.timezone`
	ServiceEnableTaskComments    Key = `service.enabletaskcomments`
	ServiceEnableTotp            Key = `service.enabletotp`
	ServiceSentryDsn             Key = `service.sentrydsn`
	ServiceTestingtoken          Key = `service.testingtoken`
	ServiceEnableEmailReminders  Key = `service.enableemailreminders`
	ServiceEnableUserDeletion    Key = `service.enableuserdeletion`
	ServiceMaxAvatarSize         Key = `service.maxavatarsize`

	AuthLocalEnabled      Key = `auth.local.enabled`
	AuthOpenIDEnabled     Key = `auth.openid.enabled`
	AuthOpenIDRedirectURL Key = `auth.openid.redirecturl`
	AuthOpenIDProviders   Key = `auth.openid.providers`

	LegalImprintURL Key = `legal.imprinturl`
	LegalPrivacyURL Key = `legal.privacyurl`

	DatabaseType                  Key = `database.type`
	DatabaseHost                  Key = `database.host`
	DatabaseUser                  Key = `database.user`
	DatabasePassword              Key = `database.password`
	DatabaseDatabase              Key = `database.database`
	DatabasePath                  Key = `database.path`
	DatabaseMaxOpenConnections    Key = `database.maxopenconnections`
	DatabaseMaxIdleConnections    Key = `database.maxidleconnections`
	DatabaseMaxConnectionLifetime Key = `database.maxconnectionlifetime`
	DatabaseSslMode               Key = `database.sslmode`
	DatabaseSslCert               Key = `database.sslcert`
	DatabaseSslKey                Key = `database.sslkey`
	DatabaseSslRootCert           Key = `database.sslrootcert`
	DatabaseTLS                   Key = `database.tls`

	TypesenseEnabled Key = `typesense.enabled`
	TypesenseURL     Key = `typesense.url`
	TypesenseAPIKey  Key = `typesense.apikey`

	MailerEnabled       Key = `mailer.enabled`
	MailerHost          Key = `mailer.host`
	MailerPort          Key = `mailer.port`
	MailerUsername      Key = `mailer.username`
	MailerPassword      Key = `mailer.password`
	MailerAuthType      Key = `mailer.authtype`
	MailerSkipTLSVerify Key = `mailer.skiptlsverify`
	MailerFromEmail     Key = `mailer.fromemail`
	MailerQueuelength   Key = `mailer.queuelength`
	MailerQueueTimeout  Key = `mailer.queuetimeout`
	MailerForceSSL      Key = `mailer.forcessl`

	RedisEnabled  Key = `redis.enabled`
	RedisHost     Key = `redis.host`
	RedisPassword Key = `redis.password`
	RedisDB       Key = `redis.db`

	LogEnabled       Key = `log.enabled`
	LogStandard      Key = `log.standard`
	LogLevel         Key = `log.level`
	LogDatabase      Key = `log.database`
	LogDatabaseLevel Key = `log.databaselevel`
	LogHTTP          Key = `log.http`
	LogEcho          Key = `log.echo`
	LogPath          Key = `log.path`
	LogEvents        Key = `log.events`
	LogEventsLevel   Key = `log.eventslevel`
	LogMail          Key = `log.mail`
	LogMailLevel     Key = `log.maillevel`

	RateLimitEnabled Key = `ratelimit.enabled`
	RateLimitKind    Key = `ratelimit.kind`
	RateLimitPeriod  Key = `ratelimit.period`
	RateLimitLimit   Key = `ratelimit.limit`
	RateLimitStore   Key = `ratelimit.store`

	FilesBasePath Key = `files.basepath`
	FilesMaxSize  Key = `files.maxsize`

	MigrationTodoistEnable             Key = `migration.todoist.enable`
	MigrationTodoistClientID           Key = `migration.todoist.clientid`
	MigrationTodoistClientSecret       Key = `migration.todoist.clientsecret`
	MigrationTodoistRedirectURL        Key = `migration.todoist.redirecturl`
	MigrationTrelloEnable              Key = `migration.trello.enable`
	MigrationTrelloKey                 Key = `migration.trello.key`
	MigrationTrelloRedirectURL         Key = `migration.trello.redirecturl`
	MigrationMicrosoftTodoEnable       Key = `migration.microsofttodo.enable`
	MigrationMicrosoftTodoClientID     Key = `migration.microsofttodo.clientid`
	MigrationMicrosoftTodoClientSecret Key = `migration.microsofttodo.clientsecret`
	MigrationMicrosoftTodoRedirectURL  Key = `migration.microsofttodo.redirecturl`

	CorsEnable  Key = `cors.enable`
	CorsOrigins Key = `cors.origins`
	CorsMaxAge  Key = `cors.maxage`

	AvatarGravaterExpiration Key = `avatar.gravatarexpiration`

	BackgroundsEnabled               Key = `backgrounds.enabled`
	BackgroundsUploadEnabled         Key = `backgrounds.providers.upload.enabled`
	BackgroundsUnsplashEnabled       Key = `backgrounds.providers.unsplash.enabled`
	BackgroundsUnsplashAccessToken   Key = `backgrounds.providers.unsplash.accesstoken`
	BackgroundsUnsplashApplicationID Key = `backgrounds.providers.unsplash.applicationid`

	KeyvalueType Key = `keyvalue.type`

	MetricsEnabled  Key = `metrics.enabled`
	MetricsUsername Key = `metrics.username`
	MetricsPassword Key = `metrics.password`

	DefaultSettingsAvatarProvider              Key = `defaultsettings.avatar_provider`
	DefaultSettingsAvatarFileID                Key = `defaultsettings.avatar_file_id`
	DefaultSettingsEmailRemindersEnabled       Key = `defaultsettings.email_reminders_enabled`
	DefaultSettingsDiscoverableByName          Key = `defaultsettings.discoverable_by_name`
	DefaultSettingsDiscoverableByEmail         Key = `defaultsettings.discoverable_by_email`
	DefaultSettingsOverdueTaskRemindersEnabled Key = `defaultsettings.overdue_tasks_reminders_enabled`
	DefaultSettingsDefaultProjectID            Key = `defaultsettings.default_project_id`
	DefaultSettingsWeekStart                   Key = `defaultsettings.week_start`
	DefaultSettingsLanguage                    Key = `defaultsettings.language`
	DefaultSettingsTimezone                    Key = `defaultsettings.timezone`
	DefaultSettingsOverdueTaskRemindersTime    Key = `defaultsettings.overdue_tasks_reminders_time`

	WebhooksEnabled        Key = `webhooks.enabled`
	WebhooksTimeoutSeconds Key = `webhooks.timeoutseconds`
	WebhooksProxyURL       Key = `webhooks.proxyurl`
	WebhooksProxyPassword  Key = `webhooks.proxypassword`
)

// GetString returns a string config value
func (k Key) GetString() string {
	return viper.GetString(string(k))
}

// GetBool returns a bool config value
func (k Key) GetBool() bool {
	return viper.GetBool(string(k))
}

// GetInt returns an int config value
func (k Key) GetInt() int {
	return viper.GetInt(string(k))
}

// GetInt64 returns an int64 config value
func (k Key) GetInt64() int64 {
	return viper.GetInt64(string(k))
}

// GetDuration returns a duration config value
func (k Key) GetDuration() time.Duration {
	return viper.GetDuration(string(k))
}

// GetStringSlice returns a string slice from a config option
func (k Key) GetStringSlice() []string {
	return viper.GetStringSlice(string(k))
}

// Get returns the raw value from a config option
func (k Key) Get() interface{} {
	return viper.Get(string(k))
}

var timezone *time.Location

// GetTimeZone returns the time zone configured for vikunja
// It is a separate function and not done through viper because that makes handling
// it way easier, especially when testing.
func GetTimeZone() *time.Location {
	if timezone == nil {
		loc, err := time.LoadLocation(ServiceTimeZone.GetString())
		if err != nil {
			fmt.Printf("Error parsing time zone: %s", err)
			os.Exit(1)
		}
		timezone = loc
	}
	return timezone
}

// Set sets a value
func (k Key) Set(i interface{}) {
	viper.Set(string(k), i)
}

// sets the default config value
func (k Key) setDefault(i interface{}) {
	viper.SetDefault(string(k), i)
}

// Tries different methods to figure out the binary folder.
// Copied and adopted from https://github.com/speedata/publisher/commit/3b668668d57edef04ea854d5bbd58f83eb1b799f
func getBinaryDirLocation() string {
	// First, check if the standard library gives us the path. This will work 99% of the time.
	ex, err := os.Executable()
	if err == nil {
		return filepath.Dir(ex)
	}

	// Then check if the binary was run with a full path and use that if that's the case.
	if strings.Contains(os.Args[0], "/") {
		binDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatal(err)
		}
		return binDir
	}

	exeSuffix := ""
	if runtime.GOOS == "windows" {
		exeSuffix = ".exe"
	}

	// All else failing, search for a vikunja binary in the current $PATH.
	// This can give wrong results.
	exeLocation, err := exec.LookPath("vikunja" + exeSuffix)
	if err != nil {
		log.Fatal(err)
	}

	return filepath.Dir(exeLocation)
}

// InitDefaultConfig sets default config values
// This is an extra function so we can call it when initializing tests without initializing the full config
func InitDefaultConfig() {
	// Service config
	random, err := random(32)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Service
	ServiceJWTSecret.setDefault(random)
	ServiceJWTTTL.setDefault(259200)      // 72 hours
	ServiceJWTTTLLong.setDefault(2592000) // 30 days
	ServiceInterface.setDefault(":3456")
	ServiceUnixSocket.setDefault("")
	ServiceFrontendurl.setDefault("")
	ServiceEnableCaldav.setDefault(true)

	ServiceRootpath.setDefault(getBinaryDirLocation())
	ServiceStaticpath.setDefault("")
	ServiceMaxItemsPerPage.setDefault(50)
	ServiceEnableMetrics.setDefault(false)
	ServiceMotd.setDefault("")
	ServiceEnableLinkSharing.setDefault(true)
	ServiceEnableRegistration.setDefault(true)
	ServiceEnableTaskAttachments.setDefault(true)
	ServiceTimeZone.setDefault("GMT")
	ServiceEnableTaskComments.setDefault(true)
	ServiceEnableTotp.setDefault(true)
	ServiceEnableEmailReminders.setDefault(true)
	ServiceEnableUserDeletion.setDefault(true)
	ServiceMaxAvatarSize.setDefault(1024)
	ServiceDemoMode.setDefault(false)

	// Auth
	AuthLocalEnabled.setDefault(true)
	AuthOpenIDEnabled.setDefault(false)

	// Database
	DatabaseType.setDefault("sqlite")
	DatabaseHost.setDefault("localhost")
	DatabaseUser.setDefault("vikunja")
	DatabasePassword.setDefault("")
	DatabaseDatabase.setDefault("vikunja")
	DatabasePath.setDefault("./vikunja.db")
	DatabaseMaxOpenConnections.setDefault(100)
	DatabaseMaxIdleConnections.setDefault(50)
	DatabaseMaxConnectionLifetime.setDefault(10000)
	DatabaseSslMode.setDefault("disable")
	DatabaseSslCert.setDefault("")
	DatabaseSslKey.setDefault("")
	DatabaseSslRootCert.setDefault("")
	DatabaseTLS.setDefault("false")

	// Typesense
	TypesenseEnabled.setDefault(false)

	// Mailer
	MailerEnabled.setDefault(false)
	MailerHost.setDefault("")
	MailerPort.setDefault("587")
	MailerUsername.setDefault("")
	MailerPassword.setDefault("")
	MailerSkipTLSVerify.setDefault(false)
	MailerFromEmail.setDefault("mail@vikunja")
	MailerQueuelength.setDefault(100)
	MailerQueueTimeout.setDefault(30)
	MailerForceSSL.setDefault(false)
	MailerAuthType.setDefault("plain")
	// Redis
	RedisEnabled.setDefault(false)
	RedisHost.setDefault("localhost:6379")
	RedisPassword.setDefault("")
	RedisDB.setDefault(0)
	// Logger
	LogEnabled.setDefault(true)
	LogStandard.setDefault("stdout")
	LogLevel.setDefault("INFO")
	LogDatabase.setDefault("off")
	LogDatabaseLevel.setDefault("WARNING")
	LogHTTP.setDefault("stdout")
	LogEcho.setDefault("off")
	LogPath.setDefault(ServiceRootpath.GetString() + "/logs")
	LogEvents.setDefault("off")
	LogEventsLevel.setDefault("INFO")
	LogMail.setDefault("off")
	LogMailLevel.setDefault("INFO")
	// Rate Limit
	RateLimitEnabled.setDefault(false)
	RateLimitKind.setDefault("user")
	RateLimitLimit.setDefault(100)
	RateLimitPeriod.setDefault(60)
	RateLimitStore.setDefault("memory")
	// Files
	FilesBasePath.setDefault("files")
	FilesMaxSize.setDefault("20MB")
	// Cors
	CorsEnable.setDefault(true)
	CorsOrigins.setDefault([]string{"*"})
	CorsMaxAge.setDefault(0)
	// Migration
	MigrationTodoistEnable.setDefault(false)
	MigrationTrelloEnable.setDefault(false)
	MigrationMicrosoftTodoEnable.setDefault(false)
	// Avatar
	AvatarGravaterExpiration.setDefault(3600)
	// Project Backgrounds
	BackgroundsEnabled.setDefault(true)
	BackgroundsUploadEnabled.setDefault(true)
	BackgroundsUnsplashEnabled.setDefault(false)
	// Key Value
	KeyvalueType.setDefault("memory")
	// Metrics
	MetricsEnabled.setDefault(false)
	// Settings
	DefaultSettingsAvatarProvider.setDefault("initials")
	DefaultSettingsOverdueTaskRemindersEnabled.setDefault(true)
	DefaultSettingsOverdueTaskRemindersTime.setDefault("9:00")
	// Webhook
	WebhooksEnabled.setDefault(true)
	WebhooksTimeoutSeconds.setDefault(30)
}

// InitConfig initializes the config, sets defaults etc.
func InitConfig() {

	// Set defaults
	InitDefaultConfig()

	// Init checking for environment variables
	viper.SetEnvPrefix("vikunja")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Just load environment variables
	_ = viper.ReadInConfig()
	log.ConfigLogger(LogEnabled.GetBool(), LogStandard.GetString(), LogPath.GetString(), LogLevel.GetString())

	// Load the config file
	viper.AddConfigPath(ServiceRootpath.GetString())
	viper.AddConfigPath("/etc/vikunja/")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Debugf("No home directory found, not using config from ~/.config/vikunja/. Error was: %s\n", err.Error())
	} else {
		viper.AddConfigPath(path.Join(homeDir, ".config", "vikunja"))
	}

	viper.AddConfigPath(".")
	viper.SetConfigName("config")

	err = viper.ReadInConfig()

	if viper.ConfigFileUsed() != "" {
		log.Infof("Using config file: %s", viper.ConfigFileUsed())

		if err != nil {
			log.Warning(err.Error())
			log.Warning("Using default config.")
		} else {
			log.ConfigLogger(LogEnabled.GetBool(), LogStandard.GetString(), LogPath.GetString(), LogLevel.GetString())
		}
	} else {
		log.Info("No config file found, using default or config from environment variables.")
	}

	if RateLimitStore.GetString() == "keyvalue" {
		RateLimitStore.Set(KeyvalueType.GetString())
	}

	if ServiceFrontendurl.GetString() != "" && !strings.HasSuffix(ServiceFrontendurl.GetString(), "/") {
		ServiceFrontendurl.Set(ServiceFrontendurl.GetString() + "/")
	}

	if AuthOpenIDRedirectURL.GetString() == "" {
		AuthOpenIDRedirectURL.Set(ServiceFrontendurl.GetString() + "auth/openid/")
	}

	if MigrationTodoistRedirectURL.GetString() == "" {
		MigrationTodoistRedirectURL.Set(ServiceFrontendurl.GetString() + "migrate/todoist")
	}

	if MigrationTrelloRedirectURL.GetString() == "" {
		MigrationTrelloRedirectURL.Set(ServiceFrontendurl.GetString() + "migrate/trello")
	}

	if MigrationMicrosoftTodoRedirectURL.GetString() == "" {
		MigrationMicrosoftTodoRedirectURL.Set(ServiceFrontendurl.GetString() + "migrate/microsoft-todo")
	}

	if DefaultSettingsTimezone.GetString() == "" {
		DefaultSettingsTimezone.Set(ServiceTimeZone.GetString())
	}

	if ServiceEnableMetrics.GetBool() {
		log.Warning("service.enablemetrics is deprecated and will be removed in a future release. Please use metrics.enable.")
		MetricsEnabled.Set(true)
	}
}

func random(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return fmt.Sprintf("%X", b), nil
}
