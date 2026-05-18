#!/usr/bin/env bash
cd "$(dirname "$0")"
go build -ldflags "-X main.sourceDir=$(pwd)" -o zaelix . 2>/dev/null
./zaelix "$@"
