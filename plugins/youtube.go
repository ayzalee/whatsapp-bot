package plugins

import (
"context"
"fmt"
"os"
"os/exec"
"path/filepath"
"strings"

"go.mau.fi/whatsmeow"
waProto "go.mau.fi/whatsmeow/proto/waE2E"
"google.golang.org/protobuf/proto"
)

func init() {
Register(&Command{
Pattern:  "play",
Category: "download",
Func:     playCmd,
})
Register(&Command{
Pattern:  "yta",
Category: "download",
Func:     ytaCmd,
})
Register(&Command{
Pattern:  "ytv",
Category: "download",
Func:     ytvCmd,
})
}

func ytBaseFlags() []string {
return []string{
"--no-playlist",
"--no-warnings", "--quiet",
"--concurrent-fragments", "4",
"--no-part",
"--remote-components", "ejs:github",
"--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
}
}

func ytCookieFlag() []string {
cookieFile := dlCookieFile
if cookieFile == "" {
if _, err := os.Stat("cookies.txt"); err == nil {
abs, _ := filepath.Abs("cookies.txt")
cookieFile = abs
}
} else {
abs, _ := filepath.Abs(cookieFile)
cookieFile = abs
}
if cookieFile != "" {
return []string{"--cookies", cookieFile}
}
return nil
}

func ytDownloadAudio(ctx *Context, query string, isSearch bool) error {
tmpDir, err := os.MkdirTemp("", "yt-*")
if err != nil {
ctx.Reply("Failed to create temp directory.")
return nil
}
defer os.RemoveAll(tmpDir)

outTemplate := filepath.Join(tmpDir, "%(title).50s.%(ext)s")

args := append([]string{
"-f", "bestaudio[ext=webm]/bestaudio/best",
"-x", "--audio-format", "mp3", "--audio-quality", "0",
}, ytBaseFlags()...)

if isSearch {
args = append(args, "--default-search", "ytsearch")
}

args = append(args, ytCookieFlag()...)
args = append(args, "-o", outTemplate, query)

// Get title first
titleArgs := append([]string{"--get-title", "--no-playlist", "--quiet"}, ytCookieFlag()...)
if isSearch {
titleArgs = append(titleArgs, "--default-search", "ytsearch")
}
titleArgs = append(titleArgs, "--remote-components", "ejs:github", query)

titleOut, _ := exec.Command("yt-dlp", titleArgs...).Output()
title := strings.TrimSpace(string(titleOut))
if title == "" {
title = query
}

ctx.Reply(fmt.Sprintf("_Downloading_ *%s*...", title))

cmd := exec.Command("yt-dlp", args...)
if _, runErr := cmd.CombinedOutput(); runErr != nil {
ctx.Reply(T().DlFailed)
return nil
}

files, err := filepath.Glob(filepath.Join(tmpDir, "*"))
if err != nil || len(files) == 0 {
ctx.Reply(T().DlNoFile)
return nil
}

fileBytes, err := os.ReadFile(files[0])
if err != nil {
ctx.Reply("Failed to read file.")
return nil
}

uploaded, err := ctx.Client.Upload(context.Background(), fileBytes, whatsmeow.MediaAudio)
if err != nil {
ctx.Reply("Failed to upload audio.")
return nil
}

contextInfo := &waProto.ContextInfo{
StanzaID:      proto.String(ctx.Event.Info.ID),
Participant:   proto.String(ctx.Event.Info.Sender.String()),
QuotedMessage: &waProto.Message{Conversation: proto.String("")},
RemoteJID:     proto.String(ctx.Event.Info.Chat.String()),
}

id := ctx.Client.GenerateMessageID()
sendQueue <- sendTask{
client: ctx.Client,
to:     ctx.Event.Info.Chat,
msg: &waProto.Message{
AudioMessage: &waProto.AudioMessage{
URL:           proto.String(uploaded.URL),
DirectPath:    proto.String(uploaded.DirectPath),
MediaKey:      uploaded.MediaKey,
FileEncSHA256: uploaded.FileEncSHA256,
FileSHA256:    uploaded.FileSHA256,
FileLength:    proto.Uint64(uint64(len(fileBytes))),
Mimetype:      proto.String("audio/mpeg"),
ContextInfo:   contextInfo,
},
},
id: id,
}
return nil
}

func ytDownloadVideo(ctx *Context, url string) error {
tmpDir, err := os.MkdirTemp("", "yt-*")
if err != nil {
ctx.Reply("Failed to create temp directory.")
return nil
}
defer os.RemoveAll(tmpDir)

outTemplate := filepath.Join(tmpDir, "%(title).50s.%(ext)s")

args := append([]string{
"-f", "best[ext=mp4]/best",
"--merge-output-format", "mp4",
}, ytBaseFlags()...)

args = append(args, ytCookieFlag()...)
args = append(args, "-o", outTemplate, url)

titleArgs := append([]string{"--get-title", "--no-playlist", "--quiet", "--remote-components", "ejs:github"}, ytCookieFlag()...)
titleArgs = append(titleArgs, url)
titleOut, _ := exec.Command("yt-dlp", titleArgs...).Output()
title := strings.TrimSpace(string(titleOut))
if title == "" {
title = url
}

ctx.Reply(fmt.Sprintf("_Downloading_ *%s*...", title))

cmd := exec.Command("yt-dlp", args...)
if _, runErr := cmd.CombinedOutput(); runErr != nil {
ctx.Reply(T().DlFailed)
return nil
}

files, err := filepath.Glob(filepath.Join(tmpDir, "*"))
if err != nil || len(files) == 0 {
ctx.Reply(T().DlNoFile)
return nil
}

fileBytes, err := os.ReadFile(files[0])
if err != nil {
ctx.Reply("Failed to read file.")
return nil
}

uploaded, err := ctx.Client.Upload(context.Background(), fileBytes, whatsmeow.MediaVideo)
if err != nil {
ctx.Reply("Failed to upload video.")
return nil
}

contextInfo := &waProto.ContextInfo{
StanzaID:      proto.String(ctx.Event.Info.ID),
Participant:   proto.String(ctx.Event.Info.Sender.String()),
QuotedMessage: &waProto.Message{Conversation: proto.String("")},
RemoteJID:     proto.String(ctx.Event.Info.Chat.String()),
}

id := ctx.Client.GenerateMessageID()
sendQueue <- sendTask{
client: ctx.Client,
to:     ctx.Event.Info.Chat,
msg: &waProto.Message{
VideoMessage: &waProto.VideoMessage{
URL:           proto.String(uploaded.URL),
DirectPath:    proto.String(uploaded.DirectPath),
MediaKey:      uploaded.MediaKey,
FileEncSHA256: uploaded.FileEncSHA256,
FileSHA256:    uploaded.FileSHA256,
FileLength:    proto.Uint64(uint64(len(fileBytes))),
Mimetype:      proto.String("video/mp4"),
ContextInfo:   contextInfo,
},
},
id: id,
}
return nil
}

// .play <song name> — search and download audio
func playCmd(ctx *Context) error {
query := strings.TrimSpace(ctx.Text)
if query == "" {
ctx.Reply("Usage: .play <song name>\n\nExample: .play night changes")
return nil
}
return ytDownloadAudio(ctx, query, true)
}

// .yta <url> — download audio from URL
func ytaCmd(ctx *Context) error {
url := strings.TrimSpace(ctx.Text)
if url == "" {
ctx.Reply("Usage: .yta <youtube url>")
return nil
}
return ytDownloadAudio(ctx, url, false)
}

// .ytv <url> — download video from URL
func ytvCmd(ctx *Context) error {
url := strings.TrimSpace(ctx.Text)
if url == "" {
ctx.Reply("Usage: .ytv <youtube url>")
return nil
}
return ytDownloadVideo(ctx, url)
}

