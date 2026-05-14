package plugins

import (
"fmt"
"strings"
)

func init() {
Register(&Command{
Pattern:  "afk",
Category: "utility",
Func: func(ctx *Context) error {
args := ctx.Args
if len(args) == 0 {
ctx.Reply(menuHeader("afk") + "on — enable AFK\noff — disable AFK\nset <message> — set custom away message")
return nil
}
sub := strings.ToLower(args[0])
// Always key AFK by ownerPhone since the bot is the owner.
userKey := ownerPhone
if userKey == "" {
userKey = ctx.Event.Info.Sender.User
}
switch sub {
case "on":
existing := getAFK(userKey)
msg := ""
if existing != nil {
msg = existing.Message
}
setAFK(userKey, msg)
ctx.Reply(T().AFKEnabled)
case "off":
if getAFK(userKey) == nil {
ctx.Reply(T().AFKNotActive)
return nil
}
clearAFK(userKey)
ctx.Reply(T().AFKOff)
case "set":
msg := strings.TrimSpace(strings.TrimPrefix(ctx.Text, args[0]))
if msg == "" {
ctx.Reply(T().AFKSetUsage)
return nil
}
setAFK(userKey, msg)
ctx.Reply(fmt.Sprintf("%s\nMessage: %s", T().AFKEnabled, msg))
default:
ctx.Reply(menuHeader("afk") + "on — enable AFK\noff — disable AFK\nset <message> — set custom away message")
}
return nil
},
})
}