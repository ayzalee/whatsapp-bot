package plugins

import (
"context"
"strings"
"time"

"go.mau.fi/whatsmeow"
"go.mau.fi/whatsmeow/types"
)

var onlineStop chan struct{}

func init() {
Register(&Command{
Pattern:  "online",
Category: "utility",
Func:     onlineCmd,
})
}

func StartAlwaysOnline(client *whatsmeow.Client) {
if onlineStop != nil {
return
}
onlineStop = make(chan struct{})
go func() {
for {
select {
case <-onlineStop:
return
default:
client.SendPresence(context.Background(), types.PresenceAvailable)
time.Sleep(30 * time.Second)
}
}
}()
}

func onlineCmd(ctx *Context) error {
arg := strings.TrimSpace(ctx.Text)
switch arg {
case "on":
if onlineStop != nil {
ctx.Reply("Already running.")
return nil
}
BotSettings.mu.Lock()
BotSettings.AlwaysOnline = true
BotSettings.mu.Unlock()
SaveSettings()
StartAlwaysOnline(ctx.Client)
ctx.Reply("Always online enabled.")
case "off":
if onlineStop == nil {
ctx.Reply("Not running.")
return nil
}
close(onlineStop)
onlineStop = nil
BotSettings.mu.Lock()
BotSettings.AlwaysOnline = false
BotSettings.mu.Unlock()
SaveSettings()
ctx.Reply("Always online disabled.")
default:
ctx.Reply("Usage: .online on / .online off")
}
return nil
}
