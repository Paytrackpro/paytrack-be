package main

import (
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/webserver"
	"flag"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Db        storage.Config   `yaml:"db"`
	WebServer webserver.Config `yaml:"webServer"`
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
