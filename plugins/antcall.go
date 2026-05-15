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
fmt.Printf("[CALL] Failed to reject call from %s: %v\n", evt.From, err)
} else {
fmt.Printf("[CALL] Auto-rejected call from %s\n", evt.From)
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
ctx.Reply("Auto-reject calls is already on.")
return nil
}
autoRejectCalls = true
ctx.Reply("Auto-reject calls enabled.")
case "off":
if !autoRejectCalls {
ctx.Reply("Auto-reject calls is already off.")
return nil
}
autoRejectCalls = false
ctx.Reply("Auto-reject calls disabled.")
default:
status := "off"
if autoRejectCalls {
status = "on"
}
ctx.Reply("*Auto Reject Calls*\nStatus: " + status + "\n\n.antcall on\n.antcall off")
}
return nil
},
})
}
