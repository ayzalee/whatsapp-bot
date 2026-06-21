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

const dlDocumentThreshold = 64 * 1024 * 1024

func init() {
Register(&Command{
Pattern:  "dl",
Category: "download",
Func:     dlCmd,
})
}

func dlCmd(ctx *Context) error {
input := strings.TrimSpace(ctx.Text)
if input == "" {
ctx.Reply(T().DlUsage)
return nil
}

tmpDir, err := os.MkdirTemp("", "dl-*")
if err != nil {
ctx.Reply("Failed to create temp directory.")
return nil
}
defer os.RemoveAll(tmpDir)

outTemplate := filepath.Join(tmpDir, "%(title).50s.%(ext)s")

var args []string
isAudio := false
isURL := strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://")

baseFlags := []string{
"--no-playlist",
"--no-warnings", "--quiet",
"--concurrent-fragments", "4",
"--no-part",
"--remote-components", "ejs:github",
"--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
}

if strings.HasPrefix(input, "mp3 ") {
isAudio = true
url := strings.TrimPrefix(input, "mp3 ")
args = append([]string{
"-x", "--audio-format", "mp3", "--audio-quality", "0",
}, baseFlags...)
args = append(args, "-o", outTemplate, url)

} else if isURL {
args = append([]string{
"-f", "best[ext=mp4]/best",
"--merge-output-format", "mp4",
}, baseFlags...)
args = append(args, "-o", outTemplate, input)

} else {
isAudio = true
args = append([]string{
"-f", "bestaudio[ext=webm]/bestaudio/best",
"-x", "--audio-format", "mp3", "--audio-quality", "0",
"--default-search", "ytsearch",
}, baseFlags...)
args = append(args, "-o", outTemplate, input)
}

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
args = append(args, "--cookies", cookieFile)
}

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

filePath := files[0]
stat, err := os.Stat(filePath)
if err != nil {
ctx.Reply("Failed to read file info.")
return nil
}
fileSize := uint64(stat.Size())

f, err := os.Open(filePath)
if err != nil {
ctx.Reply("Failed to open file.")
return nil
}
defer f.Close()

msgID := ctx.Event.Info.ID
senderJID := ctx.Event.Info.Sender.String()
chatJID := ctx.Event.Info.Chat.String()

contextInfo := &waProto.ContextInfo{
StanzaID:      proto.String(msgID),
Participant:   proto.String(senderJID),
QuotedMessage: &waProto.Message{Conversation: proto.String("")},
RemoteJID:     proto.String(chatJID),
}

id := ctx.Client.GenerateMessageID()
asDocument := fileSize > dlDocumentThreshold

switch {
case asDocument:
mediaType := whatsmeow.MediaDocument
mimetype := "video/mp4"
if isAudio {
mimetype = "audio/mpeg"
}
uploaded, err := ctx.Client.UploadReader(context.Background(), f, nil, mediaType)
if err != nil {
ctx.Reply(fmt.Sprintf("Failed to upload (file too large or network issue): %s", err.Error()))
return nil
}
sendQueue <- sendTask{
client: ctx.Client,
to:     ctx.Event.Info.Chat,
msg: &waProto.Message{
DocumentMessage: &waProto.DocumentMessage{
URL:           proto.String(uploaded.URL),
DirectPath:    proto.String(uploaded.DirectPath),
MediaKey:      uploaded.MediaKey,
FileEncSHA256: uploaded.FileEncSHA256,
FileSHA256:    uploaded.FileSHA256,
FileLength:    proto.Uint64(uploaded.FileLength),
Mimetype:      proto.String(mimetype),
FileName:      proto.String(filepath.Base(filePath)),
ContextInfo:   contextInfo,
},
},
id: id,
}

case isAudio:
uploaded, err := ctx.Client.UploadReader(context.Background(), f, nil, whatsmeow.MediaAudio)
if err != nil {
ctx.Reply(fmt.Sprintf("Failed to upload audio: %s", err.Error()))
return nil
}
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
FileLength:    proto.Uint64(uploaded.FileLength),
Mimetype:      proto.String("audio/mpeg"),
ContextInfo:   contextInfo,
},
},
id: id,
}

default:
uploaded, err := ctx.Client.UploadReader(context.Background(), f, nil, whatsmeow.MediaVideo)
if err != nil {
ctx.Reply(fmt.Sprintf("Failed to upload video: %s", err.Error()))
return nil
}
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
FileLength:    proto.Uint64(uploaded.FileLength),
Mimetype:      proto.String("video/mp4"),
JPEGThumbnail: defaultThumbnail(),
ContextInfo:   contextInfo,
},
},
id: id,
}
}

return nil
}
