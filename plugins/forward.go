package plugins

import (
"bytes"
"context"
"crypto/sha256"
"fmt"
"strings"

"go.mau.fi/whatsmeow"
waProto "go.mau.fi/whatsmeow/proto/waE2E"
"go.mau.fi/whatsmeow/types"
"google.golang.org/protobuf/proto"
)

func init() {
Register(&Command{
Pattern:  "forward",
IsSudo:   true,
Category: "owner",
Func:     forwardCmd,
})
}

func forwardCmd(ctx *Context) error {
ci := ctx.Event.Message.GetExtendedTextMessage().GetContextInfo()
if ci == nil || ci.GetQuotedMessage() == nil {
ctx.Reply(T().ForwardUsage)
return nil
}

targets := strings.Split(strings.TrimSpace(ctx.Text), ",")
if len(targets) == 0 || strings.TrimSpace(targets[0]) == "" {
ctx.Reply(T().ForwardNoTarget)
return nil
}

quoted := ci.GetQuotedMessage()

sent := 0
failed := 0
var failedList []string

for _, raw := range targets {
raw = strings.TrimSpace(raw)
if raw == "" {
continue
}
jid, err := parseForwardTarget(raw)
if err != nil {
failed++
failedList = append(failedList, raw)
continue
}

if jid.Server == types.NewsletterServer {
if err := sendToNewsletter(ctx, jid, quoted); err != nil {
failed++
failedList = append(failedList, raw+" ("+err.Error()+")")
continue
}
sent++
continue
}

msg := proto.Clone(quoted).(*waProto.Message)
id := ctx.Client.GenerateMessageID()
sendQueue <- sendTask{
client: ctx.Client,
to:     jid,
msg:    msg,
id:     id,
}
sent++
}

result := fmt.Sprintf(T().ForwardDone, sent)
if failed > 0 {
result += fmt.Sprintf(T().ForwardFailed, failed, strings.Join(failedList, ", "))
}
ctx.Reply(result)
return nil
}

func parseForwardTarget(raw string) (types.JID, error) {
if strings.Contains(raw, "@") {
return types.ParseJID(raw)
}
return types.ParseJID(raw + "@s.whatsapp.net")
}

func sendToNewsletter(ctx *Context, jid types.JID, quoted *waProto.Message) error {
var data []byte
var err error
var mediaType whatsmeow.MediaType
var mimetype, caption string
var build func(url, directPath string, sha []byte, length uint64) *waProto.Message

switch {
case quoted.GetImageMessage() != nil:
img := quoted.GetImageMessage()
data, err = ctx.Client.Download(context.Background(), img)
mediaType = whatsmeow.MediaImage
mimetype = img.GetMimetype()
caption = img.GetCaption()
build = func(url, directPath string, sha []byte, length uint64) *waProto.Message {
return &waProto.Message{
ImageMessage: &waProto.ImageMessage{
URL:        proto.String(url),
DirectPath: proto.String(directPath),
FileSHA256: sha,
FileLength: proto.Uint64(length),
Mimetype:   proto.String(mimetype),
Caption:    proto.String(caption),
},
}
}
case quoted.GetVideoMessage() != nil:
vid := quoted.GetVideoMessage()
data, err = ctx.Client.Download(context.Background(), vid)
mediaType = whatsmeow.MediaVideo
mimetype = vid.GetMimetype()
caption = vid.GetCaption()
build = func(url, directPath string, sha []byte, length uint64) *waProto.Message {
return &waProto.Message{
VideoMessage: &waProto.VideoMessage{
URL:        proto.String(url),
DirectPath: proto.String(directPath),
FileSHA256: sha,
FileLength: proto.Uint64(length),
Mimetype:   proto.String(mimetype),
Caption:    proto.String(caption),
},
}
}
case quoted.GetAudioMessage() != nil:
aud := quoted.GetAudioMessage()
data, err = ctx.Client.Download(context.Background(), aud)
mediaType = whatsmeow.MediaAudio
mimetype = aud.GetMimetype()
build = func(url, directPath string, sha []byte, length uint64) *waProto.Message {
return &waProto.Message{
AudioMessage: &waProto.AudioMessage{
URL:        proto.String(url),
DirectPath: proto.String(directPath),
FileSHA256: sha,
FileLength: proto.Uint64(length),
Mimetype:   proto.String(mimetype),
PTT:        aud.PTT,
Seconds:    aud.Seconds,
},
}
}
case quoted.GetDocumentMessage() != nil:
doc := quoted.GetDocumentMessage()
data, err = ctx.Client.Download(context.Background(), doc)
mediaType = whatsmeow.MediaDocument
mimetype = doc.GetMimetype()
build = func(url, directPath string, sha []byte, length uint64) *waProto.Message {
return &waProto.Message{
DocumentMessage: &waProto.DocumentMessage{
URL:        proto.String(url),
DirectPath: proto.String(directPath),
FileSHA256: sha,
FileLength: proto.Uint64(length),
Mimetype:   proto.String(mimetype),
FileName:   doc.FileName,
},
}
}
default:
text := quoted.GetConversation()
if text == "" {
text = quoted.GetExtendedTextMessage().GetText()
}
if text == "" {
return fmt.Errorf("unsupported message type for channel")
}
_, sendErr := ctx.Client.SendMessage(context.Background(), jid, &waProto.Message{
Conversation: proto.String(text),
})
return sendErr
}

if err != nil {
return fmt.Errorf("download failed: %w", err)
}

hash := sha256.Sum256(data)
var resp whatsmeow.UploadResponse
err = ctx.Client.DangerousInternals().RawUpload(
context.Background(),
bytes.NewReader(data),
uint64(len(data)),
hash[:],
mediaType,
true,
&resp,
)
if err != nil {
return fmt.Errorf("upload failed: %w", err)
}

msg := build(resp.URL, resp.DirectPath, hash[:], uint64(len(data)))
_, err = ctx.Client.SendMessage(context.Background(), jid, msg, whatsmeow.SendRequestExtra{
MediaHandle: resp.Handle,
})
return err
}
