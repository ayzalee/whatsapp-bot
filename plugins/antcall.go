package plugins

import (
"context"
"fmt"
"strings"

"go.mau.fi/whatsmeow"
"go.mau.fi/whatsmeow/types/events"
)

type CallHook func(client *whatsmeow.Client, evt *events.CallOffer)

var callHooks []CallHook

func RegisterCallHook(fn CallHook) { callHooks = append(callHooks, fn) }

var autoRejectCalls bool

func init() {
RegisterCallHook(func(client *whatsmeow.Client, evt *events.CallOffer) {
if !autoRejectCalls {
return
}
err := client.RejectCall(context.Background(), evt.From, evt.CallID)
if err != nil {
} else {
}
})

Register(&Command{
Pattern:  "call",
IsSudo:   true,
Category: "settings",
Func: func(ctx *Context) error {
arg := strings.ToLower(strings.TrimSpace(ctx.Text))
switch arg {
case "on":
if autoRejectCalls {
ctx.Reply(T().AntcallAlreadyOn)
return nil
}
autoRejectCalls = true
ctx.Reply(T().AntcallOn)
case "off":
if !autoRejectCalls {
ctx.Reply(T().AntcallAlreadyOff)
return nil
}
autoRejectCalls = false
ctx.Reply(T().AntcallOff)
default:
status := "off"
if autoRejectCalls {
status = "on"
}
ctx.Reply(fmt.Sprintf(T().AntcallStatus, status))
}
return nil
},
})
}
