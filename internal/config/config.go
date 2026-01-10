package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         string `yaml:"env" env-default:"local"`
	StoragePath string `yaml:"storage_path" env-default:"./storage/storage.db" env-required:"true"`
	HTTPServer  struct {
		User        string        `yaml:"user" env-required:"true"`
		Password    string        `yaml:"password" env-required:"true" env:"HTTP_SERVER_PASSWORD"`
		Address     string        `yaml:"address" env-default:"0.0.0.0:8080"`
		Timeout     time.Duration `yaml:"timeout" env-default:"10s"`
		IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
	} `yaml:"http_server"`
}

func LoadConfig(path string) *Config {
	var cfg Config

	if path == "" {
		log.Fatal("Путь к конфигу не установлен")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Fatal("Файл конфигурации не существует: %w", err)
	}

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		log.Fatal("Не удалось прочитать конфигурацию: %w", err)
	}

	return &cfg
}
