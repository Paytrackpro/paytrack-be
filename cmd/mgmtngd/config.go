package main

import (
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/webserver"
)

type Config struct {
}

func loadConfig() (*Config, error) {
	return &Config{}, nil
}

func (c *Config) dbConfig() storage.Config {
	return storage.Config{}
}

func (c *Config) webConfig() webserver.Config {
	return webserver.Config{
		Port: 6789,
	}
}
