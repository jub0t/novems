package config

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v2"
)

type ConfigStruct struct {
	Verbose    bool   `yaml:"verbose"`
	Cookie     string `yaml:"cookie"`
	WebhookURL string `yaml:"webhook_url"`
	Rate       int    `yaml:"rate_limit_time_ms"`
}

var (
	config     *ConfigStruct
	configOnce sync.Once
)

// LoadConfig reads the config file from the specified path and caches the result.
func LoadConfig(path string) (*ConfigStruct, error) {
	var err error

	configOnce.Do(func() {
		// Check if the file exists
		if _, err = os.Stat(path); os.IsNotExist(err) {
			err = fmt.Errorf("config file does not exist: %s", path)
			return
		}

		// Open the config file
		f, openErr := os.Open(path)
		if openErr != nil {
			err = fmt.Errorf("failed to open config file: %v", openErr)
			return
		}
		defer f.Close()

		// Create a new config struct and decode the YAML file
		c := &ConfigStruct{}
		decoder := yaml.NewDecoder(f)
		if decodeErr := decoder.Decode(c); decodeErr != nil {
			err = fmt.Errorf("failed to decode config: %v", decodeErr)
			return
		}

		// Cache the successfully loaded configuration
		config = c
	})

	// If any error occurred during initialization, return it
	if err != nil {
		return nil, err
	}

	return config, nil
}
