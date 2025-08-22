package config

import (
	"time"

	"github.com/cockroachdb/errors"
	"github.com/spf13/viper"
)

type Config struct {
	Addr        string        `mapstructure:"addr"`
	DatabaseURL string        `mapstructure:"db_url"`
	Brokers     []string      `mapstructure:"brokers"`
	Topics      []string      `mapstructure:"topics"`
	GroupID     string        `mapstructure:"group_id"`
	Capacity    uint64        `mapstructure:"capacity"`
	TTL         time.Duration `mapstructure:"ttl"`
	Limit       uint64        `mapstructure:"limit"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, errors.WithDetail(err, "error reading config file")
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, errors.WithDetail(err, "unable to decode into config struct")
	}

	return &cfg, nil
}
