package config

import (
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/spf13/viper"
)

func setDefaultConfig() {
	viper.SetDefault("Data.LogConfig.EnableConsole", true)
	viper.SetDefault("Data.LogConfig.ConsoleJSONFormat", false)
	viper.SetDefault("Data.LogConfig.ConsoleLevel", "debug")
	viper.SetDefault("Data.LogConfig.EnableFile", true)
	viper.SetDefault("Data.LogConfig.FileJSONFormat", true)
	viper.SetDefault("Data.LogConfig.FileLevel", "debug")
	viper.SetDefault("Data.LogConfig.FileLocation", "./neuron.log")
	viper.SetDefault("Data.Env", "prod")
	viper.SetDefault("Data.Port", "9876")
	viper.SetDefault("Data.Verbose", true)
	viper.SetDefault("Data.RunnerWaitTimeout", constants.DefaultRunnerWaitTimeout)
	viper.SetDefault("Data.GracefulTimeout", constants.DefaultGracefulTimeout)
	viper.SetDefault("Data.ShutDownDelay", constants.DefaultShutDownDelay)
}
