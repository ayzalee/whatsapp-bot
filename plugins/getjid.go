package plugins

import "fmt"

func init() {
Register(&Command{
Pattern:  "jid",
Aliases:  []string{"getjid"},
IsSudo:   true,
Category: "utility",
Func: func(ctx *Context) error {
quoted := quotedMsg(ctx)

if quoted != nil {
// Reply to a message — get sender JID
sender := ctx.Event.Info.Sender
alt := ctx.Event.Info.SenderAlt
name := ctx.Event.Info.PushName

msg := fmt.Sprintf("*Contact JID*\nName: %s\nJID: %s", name, sender.String())
if alt.User != "" {
msg += fmt.Sprintf("\nAlt: %s", alt.String())
}
ctx.Reply(msg)
} else {
// No reply — get current chat JID
ctx.Reply(ctx.Event.Info.Chat.String())
}
return nil
},
})
}
