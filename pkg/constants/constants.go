package constants

import (
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	v1 "k8s.io/api/core/v1"
)

const (
	// ServiceName OpenTelemetry service name
	ServiceName = "neuron"
	// DefaultRunnerPort runner default port
	DefaultRunnerPort = 9876
	// DefaultYamlFileName default tas yaml file name.
	DefaultYamlFileName = ".tas.yml"
	// DefaultNamespace default k8 namespace.
	DefaultNamespace = "phoenix"
	// DefaultContainerVolumePath default container mount path.
	DefaultContainerVolumePath = "/workspace-cache"
	// DefaultVaultVolumePath default vault secrets  mount path.
	DefaultVaultVolumePath = "/vault/secrets"
	// DefaultVolumeName default pv volume name.
	DefaultVolumeName = "nucleus-pv-storage"
	// DefaultSecResourceName default secrets name.
	DefaultSecResourceName = "nucleus-secrets"
	// DefaultUserRepoSecVolumeName secret volume name for user's secrets.
	DefaultUserRepoSecVolumeName = "user-repo-secret-volume"
	// DefaultUserTokVolumeName token volume name for user's oauth token.
	DefaultUserTokVolumeName = "user-token-volume"
	// DefaultPodTimeout default is nucleus pod timeout.
	DefaultPodTimeout = 3 * time.Hour
	// BuildTimeout default build timeout.
	BuildTimeout = 12 * time.Hour
	// MysqlMaxIdleConnection max mysql idle connections.
	MysqlMaxIdleConnection = 25
	// MysqlMaxOpenConnection max mysql open connections.
	MysqlMaxOpenConnection = 25
	// MysqlMaxConnectionLifetime max mysql connection lifetime.
	MysqlMaxConnectionLifetime = 5 * time.Minute
	// DefaultVaultNamespace the default vault namespace.
	DefaultVaultNamespace = "admin"
	// DefaultCredits default user credits.
	DefaultCredits = 100
	// DefaultMetricsFileName default test metrics file name.
	DefaultMetricsFileName = "metrics.csv"
	// DefaultFailureDetailsFileName is default file name containing failed tests details
	DefaultFailureDetailsFileName = "failure.json"
	// GraphVisualizationDayCount defines number of days of data returned by repository graph API.
	GraphVisualizationDayCount = 29
	// MonthInterval is used to populate graph data of last month's
	MonthInterval = 31
	// Base10 is used in parsing ints from string
	Base10 = 10
	// BitSize16 represent bitSize 16 of integers in which the result of parsing strings must fit into
	BitSize16 = 16
	// BitSize32 represent bitSize 32 of integers in which the result of parsing strings must fit into
	BitSize32 = 32
	// BitSize64 represent bitSize 64 of integers in which the result of parsing strings must fit into
	BitSize64 = 64
	// TestLocatorsDelimiter is used to separate test locators for a task.
	TestLocatorsDelimiter = "#TAS#"
	// LocatorDelimiter is used to separate each component of a locator.
	LocatorDelimiter = "##"
	// DefaultShutDownDelay is the delay for graceful shutdown of all queue consumers
	DefaultShutDownDelay = 15e9 // 15 seconds, value is int64 nanoseconds due to issue in viper.
	// DefaultRunnerWaitTimeout is the default timeout for runner to exit gracefully
	DefaultRunnerWaitTimeout = 4 * 36e11 // 4 hours
	// DefaultGracefulTimeout is default timeout for graceful shutdown of the app.
	DefaultGracefulTimeout = 5 * 6e10 // 5 minutes
	// GithubAppBaseURL API Base URL endpoint
	GithubAppBaseURL = "https://api.github.com/app"
)

// All possible env values
const (
	Dev   = "dev"
	Prod  = "prod"
	Stage = "stage"
	Pi    = "pi"
)

// MIMECSV type not present in gin
const (
	MIMECSV = "text/csv"
)

const (
	// DefaultAvatarImageURL default avatar image url for an org.
	DefaultAvatarImageURL = "https://avatars.githubusercontent.com/u/77802575"
)

// These values are used in storing user information as a key in database
const (
	L1     = "1-3 Years"
	L2     = "4-6 Years"
	L3     = "7-10 Years"
	L4     = "More than 10 Years"
	Small  = "Less than 10 members"
	Medium = "10-100 members"
	Large  = "100-500 members"
	Xlarge = "More than 500 members"
)

const (
	// RepoURLSplitSize is size of slice when repo url is split with slash as delimiter
	RepoURLSplitSize = 3
	// DefaultLayout is the default layout to parse time strings
	DefaultLayout = "2006-01-02 15:04:05 +0000 UTC"
	// FloatPrecision value is used in parsing string to float
	FloatPrecision = 64
	// BlobMetricsLength is default number of columns in metrics csv
	BlobMetricsLength = 4
	// OOMKilledExitCode is exit code for OOMKilled error.
	OOMKilledExitCode = 137
	// GithubAppJWTExpiryDuration github app jwt expiry duration
	GithubAppJWTExpiryDuration = 10 * time.Minute
	// ExpiryDelta butffer time for token expiry
	ExpiryDelta = 15 * time.Minute
)

// UserTeamSize maps team size info to db entity
var UserTeamSize map[string]string

// UserExperience maps experience info to db entity
var UserExperience map[string]string

// NucleusDockerImage default nucleus image
var NucleusDockerImage map[core.Runner]map[string]string

// NucleusImagePolicy k8 image pull policy
var NucleusImagePolicy map[string]v1.PullPolicy

// CorsAllowedOrigins list of allowed origins
var CorsAllowedOrigins = []string{"https://dev.lambdatest.io:3000",
	"https://dev.lambdatestinternal.com:3000",
	"https://stage-tas.lambdatestinternal.com",
	"https://tas.lambdatest.com",
	"https://tas-pi.lambdatest.com"}

var GitStatusLabel, CookieName, CookieDomain, StorageClassName,
	NucleusPodMinCPULimit, NucleusPodMinMemoryLimit map[string]string

//nolint:gochecknoinits
func init() {
	NucleusDockerImage = map[core.Runner]map[string]string{
		core.CloudRunner: {
			Stage: "testatscale.azurecr.io/nucleus:latest",
			Dev:   "nucleus",
			Prod:  "lambdatasprodacr.azurecr.io/nucleus:latest",
			Pi:    "preprodtas.azurecr.io/nucleus:latest",
		},
		core.SelfHosted: {
			Stage: "lambdatest/nucleus:dev",
			Dev:   "nucleus",
			Prod:  "lambdatest/nucleus:latest",
			Pi:    "lambdatest/nucleus:beta",
		},
	}

	NucleusImagePolicy =
		map[string]v1.PullPolicy{
			Stage: v1.PullAlways,
			Dev:   v1.PullNever,
			Prod:  v1.PullAlways,
			Pi:    v1.PullAlways,
		}
	NucleusPodMinCPULimit = map[string]string{
		Stage: "0.125",
		Dev:   "0.125",
		Prod:  "0.125",
		Pi:    "0.125",
	}

	NucleusPodMinMemoryLimit = map[string]string{
		Stage: "128Mi",
		Dev:   "128Mi",
		Prod:  "128Mi",
		Pi:    "128Mi",
	}

	CookieName = map[string]string{
		Stage: "_stage_session_",
		Dev:   "_dev_session_",
		Prod:  "_session_",
		Pi:    "_pi_session_",
	}

	CookieDomain = map[string]string{
		Stage: ".lambdatestinternal.com",
		Dev:   ".lambdatest.io",
		Prod:  ".lambdatest.com",
		Pi:    ".lambdatest.com",
	}

	GitStatusLabel = map[string]string{
		Stage: "test-at-scale-stage",
		Dev:   "test-at-scale-dev",
		Prod:  "test-at-scale",
		Pi:    "test-at-scale-pre-prod",
	}

	UserTeamSize = map[string]string{
		Small:  "small",
		Medium: "medium",
		Large:  "large",
		Xlarge: "xlarge",
	}

	StorageClassName = map[string]string{
		Stage: "azurefile",
		Dev:   "standard",
		Prod:  "azurefile",
		Pi:    "azurefile",
	}
	UserExperience = map[string]string{
		L1: "L1",
		L2: "L2",
		L3: "L3",
		L4: "L4",
	}
}
