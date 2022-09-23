package config

import (
	"time"

	"github.com/LambdaTest/neuron/pkg/lumber"
)

type (
	// ConfigWrapper is a wrapper for the config
	ConfigWrapper struct {
		Config `json:"data"`
	}

	// Config the application's configuration
	Config struct {
		DB                DBConfig
		Azure             Azure
		Kafka             KafkaConfig
		WebhookAddress    string `json:"webhookAddress"`
		FrontendURL       string `json:"frontendURL"`
		Port              string
		LogFile           string
		LogConfig         lumber.LoggingConfig
		Env               string
		Verbose           bool
		JWT               JWT
		Redis             Redis
		GitHub            GitHubConfig `json:"gitHubApp"`
		GitLab            GitLabConfig
		Bitbucket         BitbucketConfig
		Vault             VaultConfig
		Tracing           TracingConfig
		RunnerWaitTimeout time.Duration
		GracefulTimeout   time.Duration
		ShutDownDelay     time.Duration
		RavenRemoteHost   string
		TasTeam           map[string]string
		SenderEmail       string
	}

	// TracingConfig provides opentelemetry configurations
	TracingConfig struct {
		// OtelEndpoint for storing host name for otel collector
		OtelEndpoint string
	}

	// DBConfig providers the mysql db configuration.
	DBConfig struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}

	// Azure providers the storage configuration.
	Azure struct {
		// PayloadContainerName for storing the nucleus payloads
		PayloadContainerName string
		// CacheContainerName for storing the user's build cache
		CacheContainerName string
		// LogsContainerName for storing the user's build/task logs
		LogsContainerName string
		// StorageAccountName azure storage account name
		StorageAccountName string
		// StorageAccessKey azure storage access key
		StorageAccessKey string
		// MetricsContainerName azure metrics container name
		MetricsContainerName string
		// CoverageContainerName azure coverage container name
		CoverageContainerName string
		// CdnURL is azure cdn url to download assets
		CdnURL string
	}

	// GitHubConfig configures a GitHub authorization provider.
	GitHubConfig struct {
		// ClientID GitHub Oauth APP clientID
		ClientID string
		// ClientID GitHub Oauth APP clientSecret
		ClientSecret string
		// Scope GitHub Oauth scopes for all repo
		Scope string
		// Server GitHub Oauth APP server address
		Server string
		// PrivateKey base64 encoded github apps private key
		PrivateKey string
		// AppID Github app ID
		AppID string
		// AppName Github app Name
		AppName string
	}

	// GitLabConfig configures a GitLab authorization provider.
	GitLabConfig struct {
		// ClientID GitLab Oauth APP clientID
		ClientID string
		// ClientID GitLab Oauth APP clientSecret
		ClientSecret string
		// PrivateRepoScope GitLab Oauth scopes for private repo
		PrivateRepoScope string
		// Server GitLab Oauth APP server address
		Server string
		// RedirectURL the Oauth2 redirect URL
		RedirectURL string
		// Endpoint for refreshing access Token
		RefreshTokenEndpoint string
	}

	// BitbucketConfig configures a Bitbucket authorization provider.
	BitbucketConfig struct {
		// ClientID Bitbucket Oauth APP clientID
		ClientID string
		// ClientID Bitbucket Oauth APP clientSecret
		ClientSecret string
		// RedirectURL the Oauth2 redirect URL
		RedirectURL string
		// Endpoint for refreshing access Token
		RefreshTokenEndpoint string
	}

	// JWT represents the JWT configuration.
	JWT struct {
		// PrivateKey RSA Encoded private key
		PrivateKey string
		// PublicKey  RSA Encoded public key
		PublicKey string
		// Timeout JWT Token timeout
		Timeout time.Duration
	}
	// VaultConfig represents the vault server configuration.
	VaultConfig struct {
		// Token directly specify token(optional)
		Token string
		// Address the vault server address
		Address string
		// Namespace the vault Namespace
		Namespace string
	}
	// Redis represents the redis configuration.
	Redis struct {
		// Redis host:port address.
		Addr string
		// Redis username.
		Username string
		// Redis password.
		Password string
		// TLS enabled
		TLS bool
	}
	// KafkaConfig provides the kafka configuration.
	KafkaConfig struct {
		Brokers                   string              `json:"brokers"`
		WebhookConfig             KafkaConsumerConfig `json:"webhook"`
		TaskQueueConfig           KafkaConsumerConfig `json:"task_queue"`
		PostProcessingQueueConfig KafkaConsumerConfig `json:"post_processing_queue"`
		BuildAbortConfig          KafkaConsumerConfig `json:"build_abort_queue"`
	}
	// KafkaConsumerConfig provides the kafka configuration.
	KafkaConsumerConfig struct {
		Topic         string `json:"topic"`
		ConsumerGroup string `json:"consumer_group"`
	}
)
