SHELL := powershell.exe
.SHELLFLAGS := -NoProfile -Command

BINARY := zaelix

.PHONY: build run update-whatsmeow patch-whatsmeow

## build: compile the bot binary
build:
	go build -o $(BINARY).exe ./...

## run: run the bot
run:
	go run .

## patch-whatsmeow: re-apply local patches to ./patched (without updating version)
patch-whatsmeow:
	pwsh -NoProfile -File scripts/patch-whatsmeow.ps1

## update-whatsmeow: pull latest whatsmeow, refresh ./patched, re-apply patches, rebuild
update-whatsmeow:
	@Write-Host "Updating whatsmeow..."
	go get go.mau.fi/whatsmeow@latest
	go mod tidy
	pwsh -NoProfile -File scripts/patch-whatsmeow.ps1
	go build ./...
	@Write-Host "Done. whatsmeow updated and patched."

## help: list available targets
help:
	@Select-String "^## " Makefile | ForEach-Object { $$_.Line -replace "^## ", "" }
