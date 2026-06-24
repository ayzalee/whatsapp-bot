package plugins

import (
"context"
"fmt"
"time"

waProto "go.mau.fi/whatsmeow/proto/waE2E"
"google.golang.org/protobuf/proto"
)

func init() {
Register(&Command{
Pattern:  "gstall",
IsSudo:   true,
Category: "tools",
Func:     gstAllCmd,
})
}

func gstAllCmd(ctx *Context) error {
ci := ctx.Event.Message.GetExtendedTextMessage().GetContextInfo()
if ci == nil || ci.GetQuotedMessage() == nil {
ctx.Reply("Reply to a message with .gstall")
return nil
}
quoted := ci.GetQuotedMessage()

hasContent := quoted.GetImageMessage() != nil ||
quoted.GetVideoMessage() != nil ||
quoted.GetConversation() != "" ||
quoted.GetExtendedTextMessage() != nil

if !hasContent {
ctx.Reply("Reply to a message with .gstall")
return nil
}

groups, err := ctx.Client.GetJoinedGroups(context.Background())
if err != nil {
ctx.Reply("Failed to get groups: " + err.Error())
return nil
}
if len(groups) == 0 {
ctx.Reply("Bot is not in any groups.")
return nil
}

statusMsg := proto.Clone(quoted).(*waProto.Message)
markAsGroupStatus(statusMsg)
wrapped := &waProto.Message{
GroupStatusMessageV2: &waProto.FutureProofMessage{
Message: statusMsg,
},
}

ctx.Reply(fmt.Sprintf("Sending group status to *%d* groups...", len(groups)))

sent := 0
failed := 0
for _, g := range groups {
_, err := ctx.Client.SendMessage(context.Background(), g.JID, wrapped)
if err != nil {
failed++
} else {
sent++
}
time.Sleep(700 * time.Millisecond)
}

ctx.Reply(fmt.Sprintf("✅ Sent: %d\n❌ Failed: %d", sent, failed))
return nil
}
