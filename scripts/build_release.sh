#!/bin/bash
cd /home/cgil/cgdev/golang/go-wmts-tool/releases/
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o saveWmtsTiles-linux-amd64 ./cmd/saveWmtsTiles/main.go 
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o saveWmtsTiles-windows-amd64.exe ./cmd/saveWmtsTiles/main.go
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o saveWmtsTiles-darwin-amd64 ./cmd/saveWmtsTiles/main.go
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o saveWmtsTiles-linux-arm64 ./cmd/saveWmtsTiles/main.go
GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o saveWmtsTiles-windows-arm64.exe ./cmd/saveWmtsTiles/main.go
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o saveWmtsTiles-darwin-arm64 ./cmd/saveWmtsTiles/main.go
tar -czvf saveWmtsTiles-linux-amd64.tar.gz saveWmtsTiles-linux-amd64
tar -czvf saveWmtsTiles-darwin-amd64.tar.gz saveWmtsTiles-darwin-amd64
zip saveWmtsTiles-windows-amd64.zip saveWmtsTiles-windows-amd64.exe
tar -czvf saveWmtsTiles-linux-arm64.tar.gz saveWmtsTiles-linux-arm64
tar -czvf saveWmtsTiles-darwin-arm64.tar.gz saveWmtsTiles-darwin-arm64
zip saveWmtsTiles-windows-arm64.zip saveWmtsTiles-windows-arm64.exe
