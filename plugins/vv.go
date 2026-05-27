package plugins

import (
"context"

"go.mau.fi/whatsmeow"
waProto "go.mau.fi/whatsmeow/proto/waE2E"
"google.golang.org/protobuf/proto"
)

func init() {
Register(&Command{
Pattern:  "vv",
Category: "utility",
Func:     vvCmd,
})
}

func vvCmd(ctx *Context) error {
ci := ctx.Event.Message.GetExtendedTextMessage().GetContextInfo()
if ci == nil || ci.GetQuotedMessage() == nil {
ctx.Reply(T().VVUsage)
return nil
}

quoted := ci.GetQuotedMessage()

if img := quoted.GetImageMessage(); img != nil {
data, err := ctx.Client.Download(context.Background(), img)
if err != nil {
ctx.Reply("Failed to download: " + err.Error())
return nil
}
uploaded, err := ctx.Client.Upload(context.Background(), data, whatsmeow.MediaImage)
if err != nil {
ctx.Reply(T().VVFailed)
return nil
}
id := ctx.Client.GenerateMessageID()
sendQueue <- sendTask{
client: ctx.Client,
to:     ctx.Event.Info.Chat,
msg: &waProto.Message{
ImageMessage: &waProto.ImageMessage{
URL:           proto.String(uploaded.URL),
DirectPath:    proto.String(uploaded.DirectPath),
MediaKey:      uploaded.MediaKey,
FileEncSHA256: uploaded.FileEncSHA256,
FileSHA256:    uploaded.FileSHA256,
FileLength:    proto.Uint64(uint64(len(data))),
Mimetype:      img.Mimetype,
},
},
id: id,
}
return nil
}

if vid := quoted.GetVideoMessage(); vid != nil {
data, err := ctx.Client.Download(context.Background(), vid)
if err != nil {
ctx.Reply("Failed to download: " + err.Error())
return nil
}
uploaded, err := ctx.Client.Upload(context.Background(), data, whatsmeow.MediaVideo)
if err != nil {
ctx.Reply(T().VVFailed)
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
FileLength:    proto.Uint64(uint64(len(data))),
Mimetype:      vid.Mimetype,
},
},
id: id,
}
return nil
}

if aud := quoted.GetAudioMessage(); aud != nil {
data, err := ctx.Client.Download(context.Background(), aud)
if err != nil {
ctx.Reply("Failed to download: " + err.Error())
return nil
}
uploaded, err := ctx.Client.Upload(context.Background(), data, whatsmeow.MediaAudio)
if err != nil {
ctx.Reply(T().VVFailed)
return nil
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
FileLength:    proto.Uint64(uint64(len(data))),
Mimetype:      aud.Mimetype,
PTT:           aud.PTT,
Seconds:       aud.Seconds,
Waveform:      aud.Waveform,
},
},
id: id,
}
return nil
}

ctx.Reply(T().VVUnsupported)
return nil
}
