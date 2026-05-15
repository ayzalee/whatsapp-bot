package plugins

import (
	"fmt"
	"regexp"
	"strings"
)

// urlRegex matches HTTP/HTTPS URLs and bare www. links.
var urlRegex = regexp.MustCompile(`(?i)https?://[^\s]+|www\.[^\s]+`)

func init() {
	Register(&Command{
		Pattern:  "antilink",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			chatJID := ctx.Event.Info.Chat.String()
			args := ctx.Args

			if len(args) == 0 {
				mode := getAntilinkMode(chatJID)
				ctx.Reply(menuHeader("antilink") + fmt.Sprintf(T().AntilinkStatus, mode))
				return nil
			}

			switch strings.ToLower(args[0]) {
			case "on":
				setAntilinkMode(chatJID, "delete")
				ctx.Reply(T().AntilinkOn)
			case "off":
				setAntilinkMode(chatJID, "off")
				ctx.Reply(T().AntilinkOff)
			case "set":
				if len(args) < 2 {
					ctx.Reply(T().AntilinkSetUsage)
					return nil
				}
				switch strings.ToLower(args[1]) {
				case "delete":
					setAntilinkMode(chatJID, "delete")
					ctx.Reply(fmt.Sprintf(T().AntilinkSet, "delete"))
				case "kick":
					setAntilinkMode(chatJID, "kick")
					ctx.Reply(fmt.Sprintf(T().AntilinkSet, "kick"))
				default:
					ctx.Reply(T().AntilinkUnknownAct)
				}
			default:
				ctx.Reply(T().AntilinkUsage)
			}
			return nil
		},
	})
}
