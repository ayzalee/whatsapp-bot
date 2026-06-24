package plugins

import (
"context"
"fmt"
"io"
"net/http"
"path"
"strings"

"go.mau.fi/whatsmeow"
waProto "go.mau.fi/whatsmeow/proto/waE2E"
"google.golang.org/protobuf/proto"
)

func init() {
Register(&Command{
Pattern:  "upload",
Category: "media",
Func: func(ctx *Context) error {
url := strings.TrimSpace(ctx.Text)
if url == "" {
ctx.Reply(T().UploadUsage)
return nil
}

resp, err := http.Get(url)
if err != nil {
ctx.Reply(fmt.Sprintf(T().UploadFetchFailed, err.Error()))
return nil
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
ctx.Reply(fmt.Sprintf(T().UploadHTTPFailed, resp.StatusCode))
return nil
}

data, err := io.ReadAll(resp.Body)
if err != nil {
ctx.Reply(fmt.Sprintf(T().UploadReadFailed, err.Error()))
return nil
}

mimetype := resp.Header.Get("Content-Type")
if idx := strings.Index(mimetype, ";"); idx != -1 {
mimetype = mimetype[:idx]
}
if mimetype == "" {
mimetype = "application/octet-stream"
}

filename := path.Base(url)
if filename == "" || filename == "/" || filename == "." {
filename = "file"
}

id := ctx.Client.GenerateMessageID()

switch {
case strings.HasPrefix(mimetype, "image/"):
uploaded, err := ctx.Client.Upload(context.Background(), data, whatsmeow.MediaImage)
if err != nil {
ctx.Reply(fmt.Sprintf(T().UploadFailed, err.Error()))
return nil
}
sendQueue <- sendTask{
client: ctx.Client,
to:     ctx.Event.Info.Chat,
msg: &waProto.Message{
ImageMessage: &waProto.ImageMessage{
Mimetype:      proto.String(mimetype),
URL:           proto.String(uploaded.URL),
DirectPath:    proto.String(uploaded.DirectPath),
MediaKey:      uploaded.MediaKey,
FileEncSHA256: uploaded.FileEncSHA256,
FileSHA256:    uploaded.FileSHA256,
FileLength:    proto.Uint64(uploaded.FileLength),
},
},
id: id,
}

case strings.HasPrefix(mimetype, "video/"):
uploaded, err := ctx.Client.Upload(context.Background(), data, whatsmeow.MediaVideo)
if err != nil {
ctx.Reply(fmt.Sprintf(T().UploadFailed, err.Error()))
return nil
}
sendQueue <- sendTask{
client: ctx.Client,
to:     ctx.Event.Info.Chat,
msg: &waProto.Message{
VideoMessage: &waProto.VideoMessage{
Mimetype:      proto.String(mimetype),
URL:           proto.String(uploaded.URL),
DirectPath:    proto.String(uploaded.DirectPath),
MediaKey:      uploaded.MediaKey,
FileEncSHA256: uploaded.FileEncSHA256,
FileSHA256:    uploaded.FileSHA256,
FileLength:    proto.Uint64(uploaded.FileLength),
JPEGThumbnail: defaultThumbnail(),
},
},
id: id,
}

case strings.HasPrefix(mimetype, "audio/"):
uploaded, err := ctx.Client.Upload(context.Background(), data, whatsmeow.MediaAudio)
if err != nil {
ctx.Reply(fmt.Sprintf(T().UploadFailed, err.Error()))
return nil
}
sendQueue <- sendTask{
client: ctx.Client,
to:     ctx.Event.Info.Chat,
msg: &waProto.Message{
AudioMessage: &waProto.AudioMessage{
Mimetype:      proto.String(mimetype),
URL:           proto.String(uploaded.URL),
DirectPath:    proto.String(uploaded.DirectPath),
MediaKey:      uploaded.MediaKey,
FileEncSHA256: uploaded.FileEncSHA256,
FileSHA256:    uploaded.FileSHA256,
FileLength:    proto.Uint64(uploaded.FileLength),
},
},
id: id,
}

default:
uploaded, err := ctx.Client.Upload(context.Background(), data, whatsmeow.MediaDocument)
if err != nil {
ctx.Reply(fmt.Sprintf(T().UploadFailed, err.Error()))
return nil
}
sendQueue <- sendTask{
client: ctx.Client,
to:     ctx.Event.Info.Chat,
msg: &waProto.Message{
DocumentMessage: &waProto.DocumentMessage{
Mimetype:      proto.String(mimetype),
URL:           proto.String(uploaded.URL),
DirectPath:    proto.String(uploaded.DirectPath),
MediaKey:      uploaded.MediaKey,
FileEncSHA256: uploaded.FileEncSHA256,
FileSHA256:    uploaded.FileSHA256,
FileLength:    proto.Uint64(uploaded.FileLength),
FileName:      proto.String(filename),
},
},
id: id,
}
}
return nil
},
})
}
