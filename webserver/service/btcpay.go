package service

type BTCConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	APIKey   string `yaml:"apiKey,omitempty"`
}
