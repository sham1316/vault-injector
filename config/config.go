package config

import (
	"encoding/json"
	"flag"
	configParser "github.com/sham1316/configparser"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"sync"
)

var config *Config
var once sync.Once
var configPath *string

type UpdateInterface interface{}

type Password string

func (p Password) MarshalJSON() ([]byte, error) {
	if 0 == len(p) {
		return []byte(`""`), nil
	} else {
		return []byte(`"XXX"`), nil
	}
}

type certificate string

func (c certificate) MarshalJSON() ([]byte, error) {
	if 0 == len(c) {
		return []byte(`""`), nil
	} else {
		return []byte(`"X509 cert"`), nil
	}

}

type Config struct {
	LogLevel    string `default:"debug" env:"LOG_LEVEL"`
	DryRun      bool   `default:"false" env:"DRY_RUN"`
	InCluster   bool   `default:"true" env:"IN_CLUSTER"`
	Kubeconfig  string `default:"" env:"KUBECONFIG"`
	TokenPath   string `default:"/var/run/secrets/kubernetes.io/serviceaccount/token" env:"TOKEN_PATH"`
	VaultAddr   string `default:"https://vault-active.vault.svc.cluster.local:8200" env:"VAULT_ADDR"`
	VaultRole   string `default:"vault-secret-syncer" env:"VAULT_ROLE"`
	SecretLabel string `default:"vault-injector" env:"SECRET_LABEL"`
	SecretMap   string `default:"map.yaml" env:"SECRET_MAP"`
	Interval    int    `default:"900" env:"INTERVAL"`
	Telegram    struct {
		Channel int64    `default:"1234" env:"TELEGRAM_ALERT_CHANEL"`
		Token   Password `env:"TELEGRAM_TOKEN"`
	}
	HTTP struct {
		ADDR        string `default:":8080" env:"HTTP_ADDR"`
		RoutePrefix string `default:"" env:"HTTP_ROUTE_PREFIX"`
	}
}

func GetCfg() *Config {
	once.Do(func() {
		configPath = flag.String("config", "config.yaml", "Configuration file path")
		flag.Parse()
		config = loadConfig(configPath)
		initZap(config)
		b, _ := json.Marshal(config) //nolint:errcheck
		zap.S().Debug(string(b))
	})
	return config
}

func initZap(config *Config) *zap.Logger {
	zapCfg := zap.NewProductionConfig()
	zapCfg.DisableStacktrace = true
	zapCfg.Encoding = "console"
	zapCfg.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	logLevel, _ := zapcore.ParseLevel(config.LogLevel) //nolint:errcheck
	zapCfg.Level = zap.NewAtomicLevelAt(logLevel)
	zapLogger, _ := zapCfg.Build() //nolint:errcheck
	zap.ReplaceGlobals(zapLogger)
	return zapLogger
}

func loadConfig(configFile *string) *Config {
	config := Config{}
	_ = configParser.SetValue(&config, "default") //nolint:errcheck
	configYamlFile, _ := os.ReadFile(*configFile) //nolint:errcheck
	_ = yaml.Unmarshal(configYamlFile, &config)   //nolint:errcheck
	_ = configParser.SetValue(&config, "env")     //nolint:errcheck
	return &config
}
