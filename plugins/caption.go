package plugins

import (
"strings"

waProto "go.mau.fi/whatsmeow/proto/waE2E"
"google.golang.org/protobuf/proto"
)

func init() {
Register(&Command{
Pattern:  "caption",
Category: "media",
Func: func(ctx *Context) error {
ci := ctx.Event.Message.GetExtendedTextMessage().GetContextInfo()
if ci == nil || ci.GetQuotedMessage() == nil {
ctx.Reply(T().CaptionUsage)
return nil
}

newCaption := strings.TrimSpace(ctx.Text)
quoted := ci.GetQuotedMessage()
msg := proto.Clone(quoted).(*waProto.Message)

switch {
case msg.GetImageMessage() != nil:
msg.ImageMessage.Caption = proto.String(newCaption)
case msg.GetVideoMessage() != nil:
msg.VideoMessage.Caption = proto.String(newCaption)
case msg.GetDocumentMessage() != nil:
msg.DocumentMessage.Caption = proto.String(newCaption)
default:
ctx.Reply(T().CaptionUnsupported)
return nil
}

id := ctx.Client.GenerateMessageID()
sendQueue <- sendTask{
client: ctx.Client,
to:     ctx.Event.Info.Chat,
msg:    msg,
id:     id,
}
return nil
},
})
}
