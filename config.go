package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

type QbittorrentServer struct {
	Protocol   string `json:"protocol,omitempty"`
	Hostname   string `json:"hostname"`
	Port       string `json:"port,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	ApiVersion string `json:"api_version,omitempty"`
	Authtype   string `json:"auth_type"`

	FormattedHostname string `json:"omit"`
}

func loadConfig() ([]*QbittorrentServer, time.Duration) {
	configJSON := os.Getenv("QBITTORRENT_SERVERS")
	if configJSON == "" {
		log.Fatal("ERROR [config] QBITTORRENT_SERVERS environment variable not set")
	}

	var configs []*QbittorrentServer
	err := json.Unmarshal([]byte(configJSON), &configs)
	if err != nil {
		log.Fatalf("ERROR [config] parsing QBITTORRENT_SERVERS: %v", err)
	}

	for _, server := range configs {
		server.SetDefaults()
	}

	recheckIntervalStr := os.Getenv("QBITTORRENT_RECHECK_INTERVAL")
	recheckInterval, err := strconv.Atoi(recheckIntervalStr)
	if err != nil || recheckInterval <= 0 {
		recheckInterval = defaultRecheckInterval
	}
	recheckDuration := time.Duration(recheckInterval) * time.Second
	return configs, recheckDuration
}

func (s *QbittorrentServer) SetDefaults() {
	if s.Protocol == "" {
		s.Protocol = "https"
	}

	if s.ApiVersion == "" {
		s.ApiVersion = "v2"
	}

	if s.Port == "" {
		s.FormattedHostname = fmt.Sprintf("%s://%s", s.Protocol, s.Hostname)
	} else {
		s.FormattedHostname = fmt.Sprintf("%s://%s:%s", s.Protocol, s.Hostname, s.Port)
	}

	if s.Authtype == "" {
		s.Authtype = "none"
		s.Username = ""
		s.Password = ""
	}
}
