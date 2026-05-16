package plugins

import (
"context"
"fmt"
	"strings"
"time"

"go.mau.fi/whatsmeow/types"
)

var autoViewStatus bool
var statusViewEmojis = []string{"❤️"}

func init() {
Register(&Command{
Pattern:  "status",
IsSudo:   true,
Category: "settings",
Func: func(ctx *Context) error {
arg := strings.ToLower(strings.TrimSpace(ctx.Text))
switch arg {
case "on":
if autoViewStatus {
ctx.Reply(T().StatusAlreadyOn)
return nil
}
autoViewStatus = true
ctx.Reply(T().StatusOn)
case "off":
if !autoViewStatus {
ctx.Reply(T().StatusAlreadyOff)
return nil
}
autoViewStatus = false
ctx.Reply(T().StatusOff)
default:
status := "off"
if autoViewStatus {
status = "on"
}
ctx.Reply(fmt.Sprintf(T().StatusInfo, status))
}
return nil
},
})
}

func HandleAutoView(client interface {
MarkRead(ctx context.Context, ids []types.MessageID, timestamp time.Time, chat, sender types.JID, receiptTypeExtra ...types.ReceiptType) error
}, info interface {
GetChat() types.JID
GetID() types.MessageID
GetSender() types.JID
}) {
}
