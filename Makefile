build:
	go build -o exporter

run:
  export QBITTORRENT_SERVERS=$(jq -c . config.json)
  go run .
  