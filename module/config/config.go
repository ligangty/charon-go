package config

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
	"org.commonjava/charon/module/util"
	"org.commonjava/charon/module/util/files"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

var globalConfig *CharonConfig
var lock = &sync.Mutex{}

// CharonConfig is used to store all configurations for charon
// tools.
// The configuration file will be named as charon.yaml, and will be stored
// in $HOME/.charon/ folder by default.
type CharonConfig struct {
	AwsProfile            string               `yaml:"aws_profile"`
	AwsCFEnable           bool                 `yaml:"aws_cf_enable"`
	IgnorePatterns        []string             `yaml:"ignore_patterns"`
	Targets               map[string][]*Target `yaml:"targets"`
	ManifestBucket        string               `yaml:"manifest_bucket"`
	IgnoreSignatureSuffix map[string][]string  `yaml:"ignore_signature_suffix"`
	SignatureCommand      string               `yaml:"detach_signature_command"`
}

type Target struct {
	Bucket   string `yaml:"bucket"`
	Prefix   string `yaml:"prefix"`
	Registry string `yaml:"registry"`
	Domain   string `yaml:"domain"`
}

func (c *CharonConfig) GetTarget(t string) []*Target {
	target_ := c.Targets[t]
	if target_ == nil {
		logger.Error(fmt.Sprintf("The target %s is not found in charon configuration.", t))
	}
	return target_
}

func (c *CharonConfig) GetIgnoreSignatureSuffix(pkgType string) []string {
	xartifactList := c.IgnoreSignatureSuffix[pkgType]
	if len(xartifactList) == 0 {
		logger.Error(fmt.Sprintf("package type %s does not have ignore artifact config.", pkgType))
	}
	return xartifactList
}

func GetConfig(cfgFilePath string) (*CharonConfig, error) {
	if globalConfig != nil {
		return globalConfig, nil
	}
	lock.Lock()
	defer lock.Unlock()
	configFilePath := cfgFilePath
	if strings.TrimSpace(configFilePath) == "" || !files.FileOrDirExists(configFilePath) {
		configFilePath = path.Join(os.Getenv("HOME"), ".charon", util.CONFIG_FILE)
	}
	yamlFile, err := files.ReadFile(configFilePath)
	if err != nil {
		logger.Error(fmt.Sprintf("Can not read yaml file %s, error:  #%v ", configFilePath, err))
		return nil, err
	}
	var c CharonConfig
	err = yaml.Unmarshal([]byte(yamlFile), &c)
	if err != nil {
		logger.Error(fmt.Sprintf("Yaml Unmarshal error: %s", err))
		return nil, err
	}

	// Set default registries for missing
	for _, v := range c.Targets {
		for _, t := range v {
			if strings.TrimSpace(t.Registry) == "" {
				t.Registry = util.DEFAULT_REGISTRY
			}
		}
	}

	err = validateConfig(&c)
	if err != nil {
		return nil, err
	}

	// Store a singleton to reuse
	globalConfig = &c

	return globalConfig, nil
}

func resetGlobal() {
	lock.Lock()
	defer lock.Unlock()
	globalConfig = nil
}

func validateConfig(conf *CharonConfig) error {
	targets := conf.Targets
	const MISSING_FIELD = "'%s' is a required property"
	if len(targets) == 0 {
		return fmt.Errorf(MISSING_FIELD, "targets")
	}
	for _, v := range targets {
		for _, t := range v {
			if strings.TrimSpace(t.Bucket) == "" {
				return fmt.Errorf(MISSING_FIELD, "bucket")
			}
		}
	}
	return nil
}
