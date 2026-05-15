package plugins

import (
	"fmt"
	"strings"
)

func init() {
	Register(&Command{
		Pattern:  "antispam",
		IsGroup:  false,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			chatJID := ctx.Event.Info.Chat.String()
			args := ctx.Args

			if len(args) == 0 {
				mode := getAntispamMode(chatJID)
				ctx.Reply(menuHeader("antispam") + fmt.Sprintf(T().AntispamStatus, mode))
				return nil
			}

			switch strings.ToLower(args[0]) {
			case "on":
				setAntispamMode(chatJID, "on")
				ctx.Reply(T().AntispamOn)
			case "off":
				setAntispamMode(chatJID, "off")
				ctx.Reply(T().AntispamOff)
			case "allow":
				arg := ""
				if len(args) > 1 {
					arg = args[1]
				}
				phone, lid := ResolveTarget(ctx, arg)
				if phone == "" && lid == "" {
					ctx.Reply(T().UserResolveFail)
					return nil
				}
				userID := phone
				if userID == "" {
					userID = lid
				}
				setAntispamWhitelist(chatJID, userID, true)
				ctx.Reply(T().AntispamAllowed)
			default:
				ctx.Reply(T().AntispamUsage)
			}
			return nil
		},
	})
}
