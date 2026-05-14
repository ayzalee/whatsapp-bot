package plugins

import (
	"fmt"
	"strings"
)

func init() {
	Register(&Command{
		Pattern:  "antistatus",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			chatJID := ctx.Event.Info.Chat.String()
			sub := strings.ToLower(strings.TrimSpace(ctx.Text))
			switch sub {
			case "on":
				setAntistatusEnabled(chatJID, true)
				ctx.Reply(T().AntistatusOn)
			case "off":
				setAntistatusEnabled(chatJID, false)
				ctx.Reply(T().AntistatusOff)
			default:
				status := "off"
				if getAntistatusEnabled(chatJID) {
					status = "on"
				}
				ctx.Reply(fmt.Sprintf("Antistatus is currently: *%s*\nUsage: .antistatus on|off", status))
			}
			return nil
		},
	})
}
