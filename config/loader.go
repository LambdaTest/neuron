package config

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GlobalConfig stores the config instance for global use
var GlobalConfig *Config

// Load loads config from command instance to predefined config variables
func Load(cmd *cobra.Command) (*Config, error) {
	err := viper.BindPFlags(cmd.Flags())
	if err != nil {
		return nil, err
	}

	// default viper configs
	viper.SetEnvPrefix("NU")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// set default configs
	setDefaultConfig()

	if configFile, _ := cmd.Flags().GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName(".nu")
		viper.AddConfigPath("./")
		viper.AddConfigPath("/vault/secrets")
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Warning: No configuration file found. Proceeding with defaults")
	}

	return populateConfig(new(ConfigWrapper))
}
