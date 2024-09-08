package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

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
			Name: "qbittorrent_torrent_seed_ratio",
			Help: "Seed ratio of torrents.",
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

func init() {
	// Register metrics with Prometheus
	prometheus.MustRegister(torrentStatus)
	prometheus.MustRegister(torrentSeeded)
	prometheus.MustRegister(torrentUploaded)
}

// Torrent structure to parse Qbittorrent API response
type Torrent struct {
	Name      string  `json:"name"`
	State     string  `json:"state"`
	Tracker   string  `json:"tracker"`
	SeedRatio float64 `json:"ratio"`
	Uploaded  int64   `json:"uploaded"` // Data uploaded in bytes
}

// Create the Basic Auth header
func createBasicAuthHeader(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

// Scrape data from Qbittorrent API
func scrapeQbittorrentAPI(protocol, hostname, username, password string) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s://%s/api/v2/torrents/info", protocol, hostname), nil)
	if err != nil {
		log.Printf("Error creating request to Qbittorrent API: %v", err)
		return
	}

	// Add Basic Auth header if credentials are provided
	if username != "" && password != "" {
		req.Header.Add("Authorization", createBasicAuthHeader(username, password))
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error getting data from Qbittorrent API: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected status code from Qbittorrent API: %d", resp.StatusCode)
		return
	}

	var torrents []Torrent
	if err := json.NewDecoder(resp.Body).Decode(&torrents); err != nil {
		log.Printf("Error decoding JSON from Qbittorrent API: %v", err)
		return
	}

	// Update Prometheus metrics with new data
	for _, torrent := range torrents {
		stateValue := 0.0
		if torrent.State == "uploading" || torrent.State == "stalledUP" {
			stateValue = 1.0
		}
		torrentStatus.WithLabelValues(hostname, torrent.Name, getTrackerHostname(torrent.Tracker), torrent.State).Set(stateValue)
		torrentSeeded.WithLabelValues(hostname, torrent.Name, getTrackerHostname(torrent.Tracker)).Set(torrent.SeedRatio)
		torrentUploaded.WithLabelValues(hostname, torrent.Name, getTrackerHostname(torrent.Tracker)).Set(float64(torrent.Uploaded))
	}
}

// Helper to extract hostname from tracker URL
func getTrackerHostname(trackerURL string) string {
	parsedURL, err := url.Parse(trackerURL)
	if err != nil {
		log.Printf("Error parsing tracker URL: %v", err)
		return trackerURL // fallback to full tracker URL in case of error
	}
	return parsedURL.Hostname()
}

func main() {
	// Get Qbittorrent hostname and credentials from environment variables
	hostname := os.Getenv("QBITTORRENT_HOSTNAME")
	if hostname == "" {
		log.Fatal("QBITTORRENT_HOSTNAME environment variable not set")
	}

	protocol := os.Getenv("QBITTORRENT_API_PROTOCOL")
	if protocol == "" {
		protocol = "https" // Default to https
	}

	username := os.Getenv("QBITTORRENT_USERNAME")
	password := os.Getenv("QBITTORRENT_PASSWORD")

	// Get the recheck interval from an environment variable, default to 30 seconds if not set
	recheckIntervalStr := os.Getenv("QBITTORRENT_RECHECK_INTERVAL")
	recheckInterval, err := strconv.Atoi(recheckIntervalStr)
	if err != nil || recheckInterval <= 0 {
		recheckInterval = 30 // Default to 30 seconds
	}
	recheckDuration := time.Duration(recheckInterval) * time.Second

	// Set up Prometheus metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	// Scrape Qbittorrent API every configured interval
	go func() {
		for {
			scrapeQbittorrentAPI(protocol, hostname, username, password)
			time.Sleep(recheckDuration)
		}
	}()

	// Start HTTP server
	port := ":8080"
	fmt.Printf("Starting Prometheus exporter on %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Error starting HTTP server: %v", err)
	}
}
