package plugins

import (
"context"
	"fmt"
)

func init() {
Register(&Command{
Pattern:  "invite",
IsGroup:  true,
IsAdmin:  true,
Category: "group",
Func: func(ctx *Context) error {
code, err := ctx.Client.GetGroupInviteLink(context.Background(), ctx.Event.Info.Chat, false)
if err != nil {
ctx.Reply(T().InviteFailed)
return nil
}
ctx.Reply(fmt.Sprintf(T().InviteLink, "https://chat.whatsapp.com/"+code))
return nil
},
})
}
