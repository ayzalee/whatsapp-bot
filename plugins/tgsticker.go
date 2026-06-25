package plugins

import (
"context"
"encoding/json"
"fmt"
"io"
"net/http"
"os"
"os/exec"
"path/filepath"
"strings"
"sync"
"time"

"go.mau.fi/whatsmeow"
waProto "go.mau.fi/whatsmeow/proto/waE2E"
"google.golang.org/protobuf/proto"
)

var tgStopMu sync.Mutex
var tgStopFlags = map[string]bool{}

func init() {
Register(&Command{
Pattern:  "tg",
Category: "media",
Func:     tgStickerCmd,
})
}

type tgFile struct {
FileID   string `json:"file_id"`
FilePath string `json:"file_path"`
}

type tgSticker struct {
FileID     string `json:"file_id"`
IsAnimated bool   `json:"is_animated"`
IsVideo    bool   `json:"is_video"`
}

type tgStickerSet struct {
OK     bool `json:"ok"`
Result struct {
Name     string      `json:"name"`
Title    string      `json:"title"`
Stickers []tgSticker `json:"stickers"`
} `json:"result"`
Description string `json:"description"`
}

type tgFileResp struct {
OK     bool   `json:"ok"`
Result tgFile `json:"result"`
}

func tgIsStopped(chatJID string) bool {
tgStopMu.Lock()
defer tgStopMu.Unlock()
return tgStopFlags[chatJID]
}

func tgSetStop(chatJID string, v bool) {
tgStopMu.Lock()
defer tgStopMu.Unlock()
tgStopFlags[chatJID] = v
}

func tgStickerCmd(ctx *Context) error {
input := strings.TrimSpace(ctx.Text)
chatJID := ctx.Event.Info.Chat.String()

if strings.ToLower(input) == "stop" {
tgSetStop(chatJID, true)
ctx.Reply(T().TgStopMsg)
return nil
}

if input == "" {
ctx.Reply(T().TgUsage)
return nil
}

token := os.Getenv("TG_TOKEN")
if token == "" {
ctx.Reply(T().TgNoToken)
return nil
}

packName := input
if idx := strings.LastIndex(input, "/addstickers/"); idx != -1 {
packName = input[idx+len("/addstickers/"):]
} else if idx := strings.LastIndex(input, "/addemoji/"); idx != -1 {
packName = input[idx+len("/addemoji/"):]
}
packName = strings.TrimSpace(packName)
if packName == "" {
ctx.Reply(T().TgNoPackName)
return nil
}

setResp, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/getStickerSet?name=%s", token, packName))
if err != nil {
ctx.Reply(fmt.Sprintf(T().TgFetchFailed, err.Error()))
return nil
}
defer setResp.Body.Close()

setBody, err := io.ReadAll(setResp.Body)
if err != nil {
ctx.Reply(fmt.Sprintf(T().TgReadFailed, err.Error()))
return nil
}

var set tgStickerSet
if err := json.Unmarshal(setBody, &set); err != nil || !set.OK {
msg := T().TgInvalidPack
if set.Description != "" {
msg += " " + set.Description
}
ctx.Reply(msg)
return nil
}

if len(set.Result.Stickers) == 0 {
ctx.Reply(T().TgEmptyPack)
return nil
}

tgSetStop(chatJID, false)
ctx.Reply(fmt.Sprintf(T().TgFoundSending, len(set.Result.Stickers), set.Result.Title))

sent := 0
skipped := 0
stopped := false

for _, s := range set.Result.Stickers {
if tgIsStopped(chatJID) {
stopped = true
break
}

if s.IsAnimated && !s.IsVideo {
skipped++
continue
}

filePath, data, err := downloadTgFile(token, s.FileID)
if err != nil || len(data) == 0 {
skipped++
continue
}

var webpData []byte
isAnimated := false

if strings.HasSuffix(filePath, ".webm") {
isAnimated = true
webpData, err = convertWebmToWebp(data)
if err != nil {
skipped++
continue
}
} else {
webpData = data
}

uploaded, err := ctx.Client.Upload(context.Background(), webpData, whatsmeow.MediaImage)
if err != nil {
skipped++
continue
}

id := ctx.Client.GenerateMessageID()
sendQueue <- sendTask{
client: ctx.Client,
to:     ctx.Event.Info.Chat,
msg: &waProto.Message{
StickerMessage: &waProto.StickerMessage{
URL:           proto.String(uploaded.URL),
DirectPath:    proto.String(uploaded.DirectPath),
MediaKey:      uploaded.MediaKey,
FileEncSHA256: uploaded.FileEncSHA256,
FileSHA256:    uploaded.FileSHA256,
FileLength:    proto.Uint64(uint64(len(webpData))),
Mimetype:      proto.String("image/webp"),
IsAnimated:    proto.Bool(isAnimated),
},
},
id: id,
}
sent++
time.Sleep(400 * time.Millisecond)
}

tgSetStop(chatJID, false)

result := fmt.Sprintf(T().TgResultLine, sent)
if skipped > 0 {
result += fmt.Sprintf(T().TgSkippedLine, skipped)
}
if stopped {
result += T().TgStoppedLine
}
ctx.Reply(result)
return nil
}

func downloadTgFile(token, fileID string) (string, []byte, error) {
getFileResp, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", token, fileID))
if err != nil {
return "", nil, err
}
defer getFileResp.Body.Close()

body, err := io.ReadAll(getFileResp.Body)
if err != nil {
return "", nil, err
}

var fr tgFileResp
if err := json.Unmarshal(body, &fr); err != nil || !fr.OK {
return "", nil, fmt.Errorf("getFile failed")
}

fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", token, fr.Result.FilePath)
dlResp, err := http.Get(fileURL)
if err != nil {
return "", nil, err
}
defer dlResp.Body.Close()

data, err := io.ReadAll(dlResp.Body)
if err != nil {
return "", nil, err
}

return fr.Result.FilePath, data, nil
}

func convertWebmToWebp(data []byte) ([]byte, error) {
tmpDir, err := os.MkdirTemp("", "tgsticker-*")
if err != nil {
return nil, err
}
defer os.RemoveAll(tmpDir)

inFile := filepath.Join(tmpDir, "in.webm")
outFile := filepath.Join(tmpDir, "out.webp")

if err := os.WriteFile(inFile, data, 0644); err != nil {
return nil, err
}

cmd := exec.Command("ffmpeg", "-i", inFile,
"-vf", "scale=512:512:force_original_aspect_ratio=decrease,fps=15",
"-vcodec", "libwebp",
"-lossless", "0",
"-compression_level", "6",
"-quality", "70",
"-loop", "0",
"-preset", "default",
"-an", "-vsync", "0",
outFile,
)
if err := cmd.Run(); err != nil {
return nil, err
}

return os.ReadFile(outFile)
}
