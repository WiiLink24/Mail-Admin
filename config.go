package main

import (
	"encoding/xml"
	"os"
)

func GetConfig() *Config {
	data, err := os.ReadFile("config.xml")
	checkError(err)

	var _config Config
	err = xml.Unmarshal(data, &_config)
	checkError(err)

	return &_config
}
