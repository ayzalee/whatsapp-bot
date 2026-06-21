package plugins

import (
"context"
"fmt"
"os"
"os/exec"
"runtime"
"strconv"
"strings"

"go.mau.fi/whatsmeow"
waProto "go.mau.fi/whatsmeow/proto/waE2E"
"google.golang.org/protobuf/proto"
)

func init() {
Register(&Command{
Pattern:  "compress",
Category: "media",
Func:     compressCmd,
})
}

func compressCmd(ctx *Context) error {
ci := ctx.Event.Message.GetExtendedTextMessage().GetContextInfo()
if ci == nil || ci.GetQuotedMessage() == nil {
ctx.Reply("Reply to a video with .compress [target_mb]\n\nExample: .compress 50")
return nil
}
quoted := ci.GetQuotedMessage()

var data []byte
var err error
mimetype := "video/mp4"

switch {
case quoted.GetVideoMessage() != nil:
vid := quoted.GetVideoMessage()
data, err = ctx.Client.Download(context.Background(), vid)
mimetype = vid.GetMimetype()
case quoted.GetDocumentMessage() != nil && strings.HasPrefix(quoted.GetDocumentMessage().GetMimetype(), "video/"):
doc := quoted.GetDocumentMessage()
data, err = ctx.Client.Download(context.Background(), doc)
default:
ctx.Reply("Reply to a video to compress.")
return nil
}

if err != nil {
ctx.Reply("Failed to download: " + err.Error())
return nil
}

targetMB := int64(50)
arg := strings.TrimSpace(ctx.Text)
if arg != "" {
if v, convErr := strconv.ParseInt(arg, 10, 64); convErr == nil && v > 0 {
targetMB = v
}
}
targetBytes := targetMB * 1024 * 1024

ctx.Reply(fmt.Sprintf("Downloaded %dMB. Compressing to ~%dMB...", len(data)/1024/1024, targetMB))

srcFile, err := os.CreateTemp("", "compress-src-*.mp4")
if err != nil {
ctx.Reply("Failed to create temp file: " + err.Error())
return nil
}
srcPath := srcFile.Name()
defer os.Remove(srcPath)

if _, werr := srcFile.Write(data); werr != nil {
srcFile.Close()
ctx.Reply("Failed to write temp file: " + werr.Error())
return nil
}
srcFile.Close()

data = nil
runtime.GC()

compressedPath, err := compressVideoToFile(srcPath, targetBytes)
if err != nil {
ctx.Reply("Compression failed: " + err.Error())
return nil
}
defer os.Remove(compressedPath)

f, err := os.Open(compressedPath)
if err != nil {
ctx.Reply("Failed to open compressed file: " + err.Error())
return nil
}
defer f.Close()

uploaded, err := ctx.Client.UploadReader(context.Background(), f, nil, whatsmeow.MediaVideo)
if err != nil {
ctx.Reply("Failed to upload: " + err.Error())
return nil
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
FileLength:    proto.Uint64(uploaded.FileLength),
Mimetype:      proto.String(mimetype),
JPEGThumbnail: defaultThumbnail(),
},
},
id: id,
}
return nil
}

func probeDurationSeconds(path string) (float64, error) {
	out, err := exec.Command("ffprobe", "-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", path).Output()
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
}

func compressVideoToFile(srcPath string, targetBytes int64) (string, error) {
dstPath := srcPath + "-out.mp4"

durSec, err := probeDurationSeconds(srcPath)
if err != nil || durSec <= 0 {
durSec = 60
}

targetBitsTotal := float64(targetBytes) * 8 * 0.92
videoBitrate := int64(targetBitsTotal / durSec)
audioBitrate := int64(96000)
videoBitrate -= audioBitrate
if videoBitrate < 150000 {
videoBitrate = 150000
}

cmd := exec.Command("ffmpeg", "-y",
"-i", srcPath,
"-c:v", "libx264",
"-b:v", fmt.Sprintf("%d", videoBitrate),
"-maxrate", fmt.Sprintf("%d", videoBitrate*2),
"-bufsize", fmt.Sprintf("%d", videoBitrate*2),
"-vf", "scale='min(854,iw)':-2",
"-preset", "veryfast",
"-c:a", "aac",
"-b:a", fmt.Sprintf("%d", audioBitrate),
"-movflags", "+faststart",
dstPath,
)
if out, err := cmd.CombinedOutput(); err != nil {
return "", fmt.Errorf("%s", string(out))
}
return dstPath, nil
}
