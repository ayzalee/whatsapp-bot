package plugins

import (
"context"
"fmt"
"strings"
)

func init() {
Register(&Command{
Pattern:  "join",
IsSudo:   true,
Category: "owner",
Func: func(ctx *Context) error {
input := strings.TrimSpace(ctx.Text)
if input == "" {
ctx.Reply(T().JoinUsage)
return nil
}

code := input
if idx := strings.LastIndex(input, "chat.whatsapp.com/"); idx != -1 {
code = input[idx+len("chat.whatsapp.com/"):]
}
code = strings.TrimSpace(code)
if code == "" {
ctx.Reply(T().JoinNoCode)
return nil
}

groupJID, err := ctx.Client.JoinGroupWithLink(context.Background(), code)
if err != nil {
ctx.Reply(fmt.Sprintf(T().JoinFailed, err.Error()))
return nil
}

ctx.Reply(fmt.Sprintf(T().JoinOK, groupJID.String()))
return nil
},
})
}
