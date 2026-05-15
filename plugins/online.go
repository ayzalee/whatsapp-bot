package plugins

import (
"context"
"strings"
"time"

"go.mau.fi/whatsmeow"
"go.mau.fi/whatsmeow/types"
)

var presenceLoop context.CancelFunc

func StartOnlineLoop(client *whatsmeow.Client) {
if presenceLoop != nil {
return
}
ctx, cancel := context.WithCancel(context.Background())
presenceLoop = cancel
go func() {
ticker := time.NewTicker(25 * time.Second)
defer ticker.Stop()
_ = client.SendPresence(context.Background(), types.PresenceAvailable)
for {
select {
case <-ticker.C:
_ = client.SendPresence(context.Background(), types.PresenceAvailable)
case <-ctx.Done():
_ = client.SendPresence(context.Background(), types.PresenceUnavailable)
return
}
}
}()
}

func StopOnlineLoop() {
if presenceLoop != nil {
presenceLoop()
presenceLoop = nil
}
}

func init() {
Register(&Command{
Pattern:  "online",
IsSudo:   true,
Category: "settings",
Func: func(ctx *Context) error {
arg := strings.ToLower(strings.TrimSpace(ctx.Text))
switch arg {
case "on":
if BotSettings.IsOnlineMode() {
ctx.Reply("Already online.")
return nil
}
BotSettings.SetOnlineMode(true)
_ = SaveSettings()
StartOnlineLoop(ctx.Client)
ctx.Reply("Always-online enabled.")
case "off":
if !BotSettings.IsOnlineMode() {
ctx.Reply("Already offline.")
return nil
}
BotSettings.SetOnlineMode(false)
_ = SaveSettings()
StopOnlineLoop()
ctx.Reply("Always-online disabled.")
default:
status := "off"
if BotSettings.IsOnlineMode() {
status = "on"
}
ctx.Reply("*Online mode:* " + status + "\n\n.online on\n.online off")
}
return nil
},
})
}
