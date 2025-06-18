#!/bin/sh

ORIG_DIR=$(pwd)
cd backplane/ || exit 1
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o=coverage.html
cd "$ORIG_DIR"
