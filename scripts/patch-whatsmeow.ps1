# patch-whatsmeow.ps1
# Copies the current whatsmeow version from the module cache into ./patched/
# and applies all local patches.

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root = Split-Path $PSScriptRoot -Parent

# ── 1. Resolve version from go.mod ──────────────────────────────────────────
Push-Location $root
$modLine = go list -m go.mau.fi/whatsmeow 2>&1
Pop-Location

if ($modLine -notmatch "go\.mau\.fi/whatsmeow\s+(\S+)") {
    Write-Error "Could not determine whatsmeow version from go.mod"
    exit 1
}
$version = $Matches[1]
Write-Host "whatsmeow version: $version"

# ── 2. Ensure module is in cache ─────────────────────────────────────────────
Push-Location $root
go mod download go.mau.fi/whatsmeow@$version | Out-Null
Pop-Location

$gopath = (go env GOPATH)
$src = "$gopath\pkg\mod\go.mau.fi\whatsmeow@$version"
if (-not (Test-Path $src)) {
    Write-Error "Module cache path not found: $src"
    exit 1
}

# ── 3. Replace ./patched with fresh copy ─────────────────────────────────────
$dst = Join-Path $root "patched"
if (Test-Path $dst) {
    Write-Host "Removing old patched/ ..."
    Remove-Item $dst -Recurse -Force
}
Write-Host "Copying $src -> patched/ ..."
Copy-Item $src $dst -Recurse -Force
# Make all files writable
Get-ChildItem $dst -Recurse | ForEach-Object {
    $_.Attributes = $_.Attributes -band (-bnot [System.IO.FileAttributes]::ReadOnly)
}

# ── 4. Patch send.go — add PinInChatMessage to getEditAttribute ───────────────
$sendFile = Join-Path $dst "send.go"
$sendContent = [System.IO.File]::ReadAllText($sendFile)

if ($sendContent -notlike "*EditAttributePinInChat*") {
    # Anchor: closing of the switch + default return — unique in getEditAttribute
    # Handles both LF and CRLF by matching \r?\n
    $patched = $sendContent -replace `
        '(\r?\n\t\}(\r?\n)\treturn types\.EditAttributeEmpty(\r?\n)\})', `
        "`n`tcase msg.PinInChatMessage != nil:`n`t`treturn types.EditAttributePinInChat`$1"
    if ($patched -eq $sendContent) {
        Write-Warning "send.go: anchor not found — patch may need updating"
    } else {
        [System.IO.File]::WriteAllText($sendFile, $patched)
        Write-Host "Patched: send.go (PinInChatMessage)"
    }
} else {
    Write-Host "send.go already patched"
}

# ── 5. Patch appstate/encode.go — add BuildClearChat ─────────────────────────
$encodeFile = Join-Path $dst "appstate\encode.go"
$encodeContent = [System.IO.File]::ReadAllText($encodeFile)

$clearFunc = @'

// BuildClearChat builds an app state patch for clearing a chat's message history.
func BuildClearChat(target types.JID, lastMessageTimestamp time.Time, lastMessageKey *waCommon.MessageKey) PatchInfo {
	return PatchInfo{
		Type: WAPatchRegularHigh,
		Mutations: []MutationInfo{{
			Index:   []string{IndexClearChat, target.String(), "1", "0"},
			Version: 6,
			Value: &waSyncAction.SyncActionValue{
				ClearChatAction: &waSyncAction.ClearChatAction{
					MessageRange: newMessageRange(lastMessageTimestamp, lastMessageKey),
				},
			},
		}},
	}
}
'@

if ($encodeContent -notlike "*BuildClearChat*") {
    # Insert after BuildDeleteChat's closing brace (before newMessageRange)
    $anchor = "func newMessageRange("
    if ($encodeContent.Contains($anchor)) {
        $encodeContent = $encodeContent.Replace($anchor, $clearFunc + "`n" + $anchor)
        [System.IO.File]::WriteAllText($encodeFile, $encodeContent)
        Write-Host "Patched: appstate/encode.go (BuildClearChat)"
    } else {
        Write-Warning "appstate/encode.go: anchor not found — patch may need updating"
    }
} else {
    Write-Host "appstate/encode.go already patched"
}

Write-Host "`nAll patches applied. Run 'go build ./...' to verify."
