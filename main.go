package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	defaultRecheckInterval int    = 30 // Default to 30 seconds
	defaultPort            string = ":8080"
)

func init() {
	// Register metrics with Prometheus
	prometheus.MustRegister(torrentStatus)
	prometheus.MustRegister(torrentSeeded)
	prometheus.MustRegister(torrentUploaded)
}

// Torrent structure to parse Qbittorrent API response
type Torrent struct {
	Name     string  `json:"name"`
	State    string  `json:"state"`
	Tracker  string  `json:"tracker"`
	Ratio    float64 `json:"ratio"`
	Uploaded int64   `json:"uploaded"` // Data uploaded in bytes
}

func main() {

	configs, recheckDuration := loadConfig()

	setupMetricsEndpoint()

	// Run scrapes in a new thread so we don't block the metrics server
	go func() {

		for {
			var wg sync.WaitGroup
			for _, server := range configs {
				wg.Add(1)
				go func(server *QbittorrentServer) {
					defer wg.Done()
					err := QbittorrentScrape(server)
					if err != nil {
						log.Printf("ERROR [%s] %v", server.Hostname, err)
					}
				}(server)
			}
			wg.Wait()
			time.Sleep(recheckDuration)
		}
	}()

	// Start HTTP metrics server
	port := os.Getenv("EXPORTER_PORT")
	if port == "" {
		port = defaultPort
	}
	fmt.Printf("Starting Prometheus exporter on %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("ERROR [http] starting HTTP server: %v", err)
	}
}

// Helper to extract hostname from tracker URL
func getTrackerHostname(trackerURL string) string {
	parsedURL, err := url.Parse(trackerURL)
	if err != nil {
		log.Printf("WARNING [main] parsing tracker url: %v", err)
		return trackerURL // fallback to full tracker URL in case of error
	}
	return parsedURL.Hostname()
}
