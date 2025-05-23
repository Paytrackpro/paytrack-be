package main

import (
	"flag"
	"fmt"
	"os"

	"code.cryptopower.dev/mgmt-ng/be/email"
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/webserver"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Db        storage.Config   `yaml:"db"`
	WebServer webserver.Config `yaml:"webServer"`
	LogLevel  string           `yaml:"logLevel"`
	LogDir    string           `yaml:"logDir"`
	Mail      email.Config     `yaml:"mail"`
}

func loadConfig() (*Config, error) {
	var filePath string
	flag.StringVar(&filePath, "config", "mgmtngd.yaml", "-config=<path to config file>")
	flag.Parse()
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("open config file failed: %s", err.Error())
	}
	var conf Config
	err = yaml.Unmarshal(raw, &conf)
	return &conf, err
}
