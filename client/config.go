package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
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
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	fPath := filepath.Join(filepath.Dir(ex), "aether_config.json")
	f, err := os.ReadFile(fPath)
	if os.IsNotExist(err) {
		cfg = config{"localhost:8000", true, ""}
		data, err := json.MarshalIndent(cfg, "", "    ")
		if err != nil {
			log.Fatal(err)
		}
		err = os.WriteFile(fPath, data, 0644)
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
