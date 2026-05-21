package plugins

import (
"context"
"fmt"
"strings"

"go.mau.fi/whatsmeow"
)

func init() {
Register(&Command{
Pattern:  "chcreate",
IsSudo:   true,
Category: "owner",
Func: func(ctx *Context) error {
input := strings.TrimSpace(ctx.Text)
if input == "" {
ctx.Reply(T().ChCreateUsage)
return nil
}

var name, description string
parts := strings.SplitN(input, "|", 2)
name = strings.TrimSpace(parts[0])
if len(parts) > 1 {
description = strings.TrimSpace(parts[1])
}

if name == "" {
ctx.Reply(T().ChCreateUsage)
return nil
}

ctx.Reply(fmt.Sprintf(T().ChCreating, name))

_ = ctx.Client.AcceptTOSNotice(context.Background(), "20601218", "5")

params := whatsmeow.CreateNewsletterParams{
Name:        name,
Description: description,
}

newsletter, err := ctx.Client.CreateNewsletter(context.Background(), params)
if err != nil {
ctx.Reply(fmt.Sprintf(T().ChCreateFailed, err.Error()))
return nil
}

ctx.Reply(fmt.Sprintf(T().ChCreateOK,
newsletter.ThreadMeta.Name.Text,
newsletter.ID.String(),
newsletter.ThreadMeta.InviteCode,
))
return nil
},
})
}
