package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	DbFilepath     string         `yaml:"db_filepath"`
	Server         Server         `yaml:"server"`
	Service        Service        `yaml:"service"`
	StorageManager StorageManager `yaml:"storage_manager"`
	CacheManager   CacheManager   `yaml:"cache_manager"`
}

type Server struct {
	Port            int `yaml:"port"`
	ShutdownTimeout int `yaml:"shutdown_timeout"`
}

type Service struct {
	ChannelSize int    `yaml:"channel_size"`
	Workers     int    `yaml:"workers"`
	Attempts    int    `yaml:"attempts"`
	OutputDir   string `yaml:"output_dir"`
}

type StorageManager struct {
	LogChannelSize  int `yaml:"log_channel_size"`
	FileChannelSize int `yaml:"file_channel_size"`
}

type CacheManager struct {
	Size       int `yaml:"size"`
	Expiration int `yaml:"expiration"`
}

// MustInit read config file and parse it into struct.
// Panics if any operations fail.
func MustInit(filePath string) *Config {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("Error while reading config file %q: %v", filePath, err))
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(content, cfg); err != nil {
		panic(fmt.Sprintf("Error while unmarshalling configuration: %v", err))
	}
	return cfg
}
