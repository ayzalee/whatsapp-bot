package plugins

import (
"context"
	"fmt"
"strings"
"time"

"go.mau.fi/whatsmeow/types"
"go.mau.fi/whatsmeow/types/events"
"go.mau.fi/whatsmeow"
)

var autoReadMessages bool

func init() {
Register(&Command{
Pattern:  "read",
IsSudo:   true,
Category: "settings",
Func: func(ctx *Context) error {
arg := strings.ToLower(strings.TrimSpace(ctx.Text))
switch arg {
case "on":
if autoReadMessages {
ctx.Reply(T().ReadAlreadyOn)
return nil
}
autoReadMessages = true
ctx.Reply(T().ReadOn)
case "off":
if !autoReadMessages {
ctx.Reply(T().ReadAlreadyOff)
return nil
}
autoReadMessages = false
ctx.Reply(T().ReadOff)
default:
status := "off"
if autoReadMessages {
status = "on"
}
ctx.Reply(fmt.Sprintf(T().ReadStatus, status))
}
return nil
},
})
}

func HandleAutoRead(client *whatsmeow.Client, evt *events.Message) {
if !autoReadMessages {
return
}
if evt.Info.Chat == types.StatusBroadcastJID {
return
}
client.MarkRead(context.Background(), []types.MessageID{evt.Info.ID}, time.Now(), evt.Info.Chat, evt.Info.Sender)
}
