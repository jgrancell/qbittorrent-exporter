package main

import (
	"time"

	qbittorrentapi "github.com/jgrancell/qbittorrent-api"
)

// scrapeQbittorrentAPI scrapes the Qbittorrent API for the given server configuration.
// It sends an HTTP GET request to the server's API endpoint, decodes the JSON response,
// and updates Prometheus metrics with the data.
//
// Parameters:
// - server: The QbittorrentServer configuration.
//
// Returns:
// - error: An error if the request, response, or JSON decoding fails.
func QbittorrentScrape(server *QbittorrentServer) error {
	c := qbittorrentapi.QbittorrentClient{
		FQDN:    server.FormattedHostname,
		Version: server.ApiVersion,
		HttpConfig: qbittorrentapi.HttpConfig{
			Timeout: 10 * time.Second,
		},
		AuthConfig: &qbittorrentapi.AuthConfig{
			Method:   server.Authtype,
			Username: server.Username,
			Password: server.Password,
		},
	}

	info, err := c.Info()
	if err != nil {
		return err
	}

	var torrents []Torrent
	for _, torrent := range info {
		t := Torrent{
			Name:     torrent.Name,
			State:    torrent.State,
			Tracker:  torrent.Tracker,
			Ratio:    torrent.Ratio,
			Uploaded: torrent.Uploaded,
		}
		torrents = append(torrents, t)
	}

	// Update Prometheus metrics with new data
	updateMetrics(server.Hostname, torrents)
	return nil
}
