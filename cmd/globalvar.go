package cmd

import (
	log "github.com/sirupsen/logrus"
	"net/http"
)

type global struct {
	ConfigPath string
	Loglevel   log.Level
	LogPath    string
	Config     config
	HttpClient *http.Client
}

var Global = global{
	ConfigPath: "./healcheck.toml",
	Loglevel:   log.InfoLevel,
	LogPath:    "./healcheck.log",
	HttpClient: http.DefaultClient,
}

var Logger = log.New()
