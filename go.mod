module github.com/LambdaTest/neuron

go 1.17

replace github.com/drone/go-scm => github.com/LambdaTest/go-scm v1.14.4

require (
	github.com/Azure/azure-pipeline-go v0.2.3 // indirect
	github.com/Azure/azure-storage-blob-go v0.13.0
	github.com/avast/retry-go/v4 v4.0.2
	github.com/drone/go-login v1.0.3
	github.com/drone/go-scm v1.15.0
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-contrib/pprof v1.3.0
	github.com/gin-gonic/gin v1.7.7
	github.com/go-redis/redis/v8 v8.11.3
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang-jwt/jwt/v4 v4.0.0
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.2.0
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/vault/api v1.1.1
	github.com/jmoiron/sqlx v1.3.4
	github.com/json-iterator/go v1.1.12
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/pelletier/go-toml v1.9.4 // indirect
	github.com/segmentio/kafka-go v0.4.22
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/cobra v1.3.0
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.10.1
	github.com/stretchr/testify v1.7.0
	go.uber.org/atomic v1.8.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.20.0
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
	golang.org/x/net v0.0.0-20210813160813-60bc85c4be6d // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20220111092808-5a964db01320 // indirect
	golang.org/x/term v0.0.0-20210503060354-a79de5458b56 // indirect
	golang.org/x/time v0.0.0-20210608053304-ed9ce3a009e4 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/guregu/null.v4 v4.0.0
	gopkg.in/ini.v1 v1.66.2 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
	k8s.io/klog/v2 v2.40.0 // indirect
	k8s.io/utils v0.0.0-20210527160623-6fdb442a123b // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.1 // indirect
)

require (
	github.com/LambdaTest/test-at-scale v0.3.4-0.20220502065627-e85586215920
	github.com/gocraft/dbr v0.0.0-20190714181702-8114670a83bd
	github.com/pkg/errors v0.9.1
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.28.0
	go.opentelemetry.io/otel v1.3.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.3.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.3.0
	go.opentelemetry.io/otel/sdk v1.3.0
)

require (
	github.com/AlecAivazis/survey/v2 v2.2.9 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.1.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.2 // indirect
	github.com/andygrunwald/go-jira v1.13.0 // indirect
	github.com/bmatcuk/doublestar/v4 v4.0.2 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/git-chglog/git-chglog v0.14.2 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kyokomi/emoji/v2 v2.2.8 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/goveralls v0.0.8 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mitchellh/copystructure v1.1.2 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/trivago/tgo v1.0.7 // indirect
	github.com/tsuyoshiwada/go-gitcmd v0.0.0-20180205145712-5f1f5f9475df // indirect
	github.com/urfave/cli/v2 v2.3.0 // indirect
	golang.org/x/tools v0.1.7 // indirect
)

require (
	github.com/alphayan/redisqueue/v3 v3.0.2
	github.com/cenkalti/backoff/v3 v3.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.1.2 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-logr/logr v1.2.1 // indirect
	github.com/go-logr/stdr v1.2.0 // indirect
	github.com/go-playground/locales v0.14.0
	github.com/go-playground/universal-translator v0.18.0
	github.com/go-playground/validator/v10 v10.10.0
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/hashicorp/go-retryablehttp v0.6.6 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/vault/sdk v0.2.1 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/klauspost/compress v1.11.13 // indirect
	github.com/mattn/go-ieproxy v0.0.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pierrec/lz4 v2.6.0+incompatible // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	github.com/ugorji/go/codec v1.2.6 // indirect
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.3.0 // indirect
	go.opentelemetry.io/otel/trace v1.3.0 // indirect
	go.opentelemetry.io/proto/otlp v0.11.0 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa // indirect
	google.golang.org/grpc v1.43.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/square/go-jose.v2 v2.5.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)
