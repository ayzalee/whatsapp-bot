package plugins

import (
"context"
"fmt"
"strings"

waProto "go.mau.fi/whatsmeow/proto/waE2E"
"go.mau.fi/whatsmeow/types"
"google.golang.org/protobuf/proto"
)

func init() {
Register(&Command{
Pattern:  "gst",
Category: "tools",
Func:     gstCmd,
})
}

func gstCmd(ctx *Context) error {
ci := ctx.Event.Message.GetExtendedTextMessage().GetContextInfo()
if ci == nil || ci.GetQuotedMessage() == nil {
ctx.Reply("Reply to a message with .gst")
return nil
}
quoted := ci.GetQuotedMessage()

hasContent := quoted.GetImageMessage() != nil ||
quoted.GetVideoMessage() != nil ||
quoted.GetConversation() != "" ||
quoted.GetExtendedTextMessage() != nil

if !hasContent {
ctx.Reply("Reply to a message with .gst")
return nil
}

input := strings.TrimSpace(ctx.Text)
var targets []types.JID

if input == "" {
if ctx.Event.Info.Chat.Server != types.GroupServer {
ctx.Reply("Send group JID with command")
return nil
}
targets = append(targets, ctx.Event.Info.Chat)
} else {
for _, raw := range strings.Split(input, ",") {
raw = strings.TrimSpace(raw)
if raw == "" {
continue
}
jid, err := parseGstTarget(raw)
if err != nil || jid.Server != types.GroupServer {
continue
}
targets = append(targets, jid)
}
}

if len(targets) == 0 {
ctx.Reply("No valid group JIDs provided.")
return nil
}

statusMsg := proto.Clone(quoted).(*waProto.Message)
markAsGroupStatus(statusMsg)

wrapped := &waProto.Message{
GroupStatusMessageV2: &waProto.FutureProofMessage{
Message: statusMsg,
},
}

success := 0
for _, jid := range targets {
_, err := ctx.Client.SendMessage(context.Background(), jid, wrapped)
if err == nil {
success++
}
}

ctx.Reply(fmt.Sprintf("✅ Updated in %d group(s).", success))
return nil
}

func parseGstTarget(raw string) (types.JID, error) {
	if strings.Contains(raw, "@") {
		return types.ParseJID(raw)
	}
	return types.ParseJID(raw + "@g.us")
}

func markAsGroupStatus(msg *waProto.Message) {
mark := func(ci **waProto.ContextInfo) {
if *ci == nil {
*ci = &waProto.ContextInfo{}
}
(*ci).IsGroupStatus = proto.Bool(true)
}

switch {
case msg.GetImageMessage() != nil:
mark(&msg.ImageMessage.ContextInfo)
case msg.GetVideoMessage() != nil:
mark(&msg.VideoMessage.ContextInfo)
case msg.GetExtendedTextMessage() != nil:
mark(&msg.ExtendedTextMessage.ContextInfo)
case msg.GetConversation() != "":
text := msg.GetConversation()
msg.Conversation = nil
msg.ExtendedTextMessage = &waProto.ExtendedTextMessage{
Text: proto.String(text),
}
mark(&msg.ExtendedTextMessage.ContextInfo)
}
}
