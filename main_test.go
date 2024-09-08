package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
)

const mockAPIResponse = `
[
    {
        "name": "Torrent 1",
        "state": "uploading",
        "tracker": "https://tracker1.example.com/announce/foobar",
        "ratio": 1.5,
        "uploaded": 104857600
    },
    {
        "name": "Torrent 2",
        "state": "pausedUP",
        "tracker": "https://tracker2.example.com/announce/fizzbuzz",
        "ratio": 0.8,
        "uploaded": 52428800
    }
]
`

// Mock the Qbittorrent API using httptest
func mockQbittorrentServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/torrents/info" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockAPIResponse))
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))
}

// Initialize a custom Prometheus registry and metrics
func setupTestRegistry() *prometheus.Registry {
	registry := prometheus.NewRegistry()

	// Create and register new instances of the metrics for the custom registry
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

	registry.MustRegister(torrentStatus)
	registry.MustRegister(torrentSeeded)
	registry.MustRegister(torrentUploaded)

	return registry
}

// Helper function to count metrics
func countMetrics(metrics []*dto.MetricFamily) int {
	count := 0
	for _, metricFamily := range metrics {
		count += len(metricFamily.GetMetric())
	}
	return count
}

// Test scrapeQbittorrentAPI to ensure it correctly processes the mock data
func TestScrapeQbittorrentAPI(t *testing.T) {
	// Set up the custom Prometheus registry for testing
	registry := setupTestRegistry()

	// Start a mock Qbittorrent server
	server := mockQbittorrentServer()
	defer server.Close()

	// Perform the scrape
	scrapeQbittorrentAPI("http", server.URL[7:]) // Use mocked server's hostname

	// Gather the metrics from the custom registry
	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	// Count the metrics for each type
	if count := countMetrics(metrics); count != 6 {
		t.Fatalf("unexpected number of total metrics collected, got: %d, want: 6", count)
	}

	// Test specific values for a torrent
	metricValue := testutil.ToFloat64(torrentStatus.WithLabelValues(server.URL[7:], "Torrent 1", "tracker1.example.com", "uploading"))
	if metricValue != 1 {
		t.Errorf("Expected torrentStatus to be 1 for Torrent 1, got %v", metricValue)
	}

	metricValue = testutil.ToFloat64(torrentSeeded.WithLabelValues(server.URL[7:], "Torrent 1", "tracker1.example.com"))
	if metricValue != 1.5 {
		t.Errorf("Expected torrentSeeded to be 1.5 for Torrent 1, got %v", metricValue)
	}

	metricValue = testutil.ToFloat64(torrentUploaded.WithLabelValues(server.URL[7:], "Torrent 1", "tracker1.example.com"))
	if metricValue != 104857600 {
		t.Errorf("Expected torrentUploaded to be 104857600 for Torrent 1, got %v", metricValue)
	}
}

// Test the main loop and server initialization
func TestMainLoopSingleScrape(t *testing.T) {
	// Set up the custom Prometheus registry for testing
	registry := setupTestRegistry()

	// Mock server and setup
	server := mockQbittorrentServer()
	defer server.Close()

	// Override environment variables
	os.Setenv("QBITTORRENT_HOSTNAME", server.URL[7:])
	os.Setenv("QBITTORRENT_API_PROTOCOL", "http")
	os.Setenv("QBITTORRENT_RECHECK_INTERVAL", "1")

	// Directly call the scraper instead of relying on the loop
	scrapeQbittorrentAPI("http", server.URL[7:])

	// Gather the metrics from the custom registry
	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	// Count the metrics for each type
	if count := countMetrics(metrics); count != 6 {
		t.Fatalf("unexpected number of total metrics collected, got: %d, want: 6", count)
	}
}
