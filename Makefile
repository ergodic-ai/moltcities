.PHONY: build test run clean deploy

# Build all binaries
build:
	go build -o moltcities-server ./cmd/server
	go build -o moltcities ./cmd/moltcities

# Run tests
test:
	go test -v ./...

# Run server locally
run: build
	./moltcities-server

# Clean build artifacts
clean:
	rm -f moltcities-server moltcities
	rm -f *.db *.db-wal *.db-shm
	rm -f moltcities.json
	rm -rf dist/

# Build for all platforms
build-all:
	mkdir -p dist
	GOOS=darwin GOARCH=amd64 go build -o dist/moltcities-darwin-amd64 ./cmd/moltcities
	GOOS=darwin GOARCH=arm64 go build -o dist/moltcities-darwin-arm64 ./cmd/moltcities
	GOOS=linux GOARCH=amd64 go build -o dist/moltcities-linux-amd64 ./cmd/moltcities
	GOOS=windows GOARCH=amd64 go build -o dist/moltcities-windows-amd64.exe ./cmd/moltcities

# Deploy to Fly.io
deploy:
	fly deploy

# Create Fly.io app (first time only)
fly-init:
	fly apps create moltcities
	fly volumes create moltcities_data --size 1 --region sjc

# View logs
logs:
	fly logs
