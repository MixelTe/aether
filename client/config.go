package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
)

type config struct {
	Host   string `json:"host"`
	Usewss bool   `json:"usewss"`
	Secret string `json:"secret"`
}

func loadConfig() config {
	cfg := _loadConfig()
	if cfg.Host == "" {
		log.Fatal("Hostname is empty")
	}
	if strings.Contains(cfg.Host, "/") {
		log.Fatal("Hostname must not contain slashes")
	}
	return cfg
}

func _loadConfig() (cfg config) {
	const configFile = "aether_config.json"
	f, err := os.ReadFile(configFile)
	if os.IsNotExist(err) {
		cfg = config{"localhost:8000", true, ""}
		data, err := json.MarshalIndent(cfg, "", "    ")
		if err != nil {
			log.Fatal(err)
		}
		err = os.WriteFile(configFile, data, 0644)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(f, &cfg)
	if err != nil {
		log.Fatal(err)
	}
	return
}
