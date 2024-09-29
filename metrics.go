package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Define Prometheus metrics
	torrentStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "qbittorrent_torrent_status",
			Help: "Current status of torrents (e.g., 1 for active, 0 for inactive).",
		},
		[]string{"host", "name", "tracker", "state"},
	)

	torrentSeeded = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "qbittorrent_torrent_ratio",
			Help: "Seed ratio of torrents.",
		},
		[]string{"host", "name", "tracker"},
	)

	torrentSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "qbittorrent_torrent_size_bytes",
			Help: "Size of the torrent's data, in bytes",
		},
		[]string{"host", "name", "tracker"},
	)

	torrentUploaded = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "qbittorrent_torrent_uploaded_bytes",
			Help: "Total data uploaded in bytes per torrent.",
		},
		[]string{"host", "name", "tracker"},
	)
)

func setupMetricsEndpoint() {
	http.Handle("/metrics", promhttp.Handler())
}

// updateMetrics updates the Prometheus metrics with the latest data from the Qbittorrent API.
// It iterates over the list of torrents and updates the status, seed ratio, and uploaded bytes metrics.
//
// Parameters:
// - hostname: The hostname of the Qbittorrent server.
// - torrents: A slice of Torrent structs containing the latest torrent data.
func updateMetrics(hostname string, torrents []Torrent) {
	for _, torrent := range torrents {
		stateValue := 0.0
		if torrent.State == "uploading" || torrent.State == "stalledUP" {
			stateValue = 1.0
		}
		torrentStatus.WithLabelValues(hostname, torrent.Name, getTrackerHostname(torrent.Tracker), torrent.State).Set(stateValue)
		torrentSeeded.WithLabelValues(hostname, torrent.Name, getTrackerHostname(torrent.Tracker)).Set(torrent.Ratio)
		torrentUploaded.WithLabelValues(hostname, torrent.Name, getTrackerHostname(torrent.Tracker)).Set(float64(torrent.Uploaded))
		torrentSize.WithLabelValues(hostname, torrent.Name, getTrackerHostname(torrent.Tracker)).Set(float64(torrent.Size))
	}
}
