package plugins

import (
	"context"
	"fmt"
	"strings"
)

func init() {
	Register(&Command{
		Pattern:  "shh",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			chatJID := ctx.Event.Info.Chat.String()
			args := ctx.Args

			if len(args) == 0 {
				ctx.Reply(menuHeader("shh") + T().ShhUsage)
				return nil
			}

			
			if strings.ToLower(args[0]) == "off" {
				arg := ""
				if len(args) > 1 {
					arg = args[1]
				}
				phone, lid := ResolveTarget(ctx, arg)
				if phone == "" && lid == "" {
					if arg == "" {
						ctx.Reply(T().ShhOffUsage)
					} else {
						ctx.Reply(T().UserResolveFail)
					}
					return nil
				}

				group, err := ctx.Client.GetGroupInfo(context.Background(), ctx.Event.Info.Chat)
				if err != nil {
					ctx.Reply(fmt.Sprintf(T().GroupInfoFailed, err.Error()))
					return nil
				}
				p := findParticipant(group.Participants, phone, lid)
				if p == nil {
					ctx.Reply(T().UserNotFound)
					return nil
				}
				userID := p.JID.User
				if !isShhed(chatJID, userID) {
					ctx.Reply(T().ShhNotShhed)
					return nil
				}
				setUnShh(chatJID, userID)
				senderJIDStr := p.JID.ToNonAD().String()
				sendMention(ctx, fmt.Sprintf(T().ShhOffOK, "@"+userID), []string{senderJIDStr})
				return nil
			}

			
			arg0 := ""
			if len(args) > 0 {
				arg0 = args[0]
			}
			phone, lid := ResolveTarget(ctx, arg0)
			if phone == "" && lid == "" {
				if arg0 == "" {
					ctx.Reply(T().ShhUsage)
				} else {
					ctx.Reply(T().UserResolveFail)
				}
				return nil
			}

			group, err := ctx.Client.GetGroupInfo(context.Background(), ctx.Event.Info.Chat)
			if err != nil {
				ctx.Reply(fmt.Sprintf(T().GroupInfoFailed, err.Error()))
				return nil
			}
			p := findParticipant(group.Participants, phone, lid)
			if p == nil {
				ctx.Reply(T().UserNotFound)
				return nil
			}
			userID := p.JID.User
			if isShhed(chatJID, userID) {
				ctx.Reply(T().ShhAlready)
				return nil
			}
			setShh(chatJID, userID)
			senderJIDStr := p.JID.ToNonAD().String()
			sendMention(ctx, fmt.Sprintf(T().ShhOK, "@"+userID), []string{senderJIDStr})
			return nil
		},
	})
}
