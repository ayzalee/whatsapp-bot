package plugins

import (
"context"
	"fmt"
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
ctx.Reply(T().OnlineAlready)
return nil
}
BotSettings.SetOnlineMode(true)
_ = SaveSettings()
StartOnlineLoop(ctx.Client)
ctx.Reply(T().OnlineOn)
case "off":
if !BotSettings.IsOnlineMode() {
ctx.Reply(T().OfflineAlready)
return nil
}
BotSettings.SetOnlineMode(false)
_ = SaveSettings()
StopOnlineLoop()
ctx.Reply(T().OnlineOff)
default:
status := "off"
if BotSettings.IsOnlineMode() {
status = "on"
}
ctx.Reply(fmt.Sprintf(T().OnlineStatus, status))
}
return nil
},
})
}
