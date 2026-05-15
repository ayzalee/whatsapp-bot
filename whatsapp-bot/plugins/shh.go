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

			// "off user" → unshh
			if strings.ToLower(args[0]) == "off" {
				arg := ""
				if len(args) > 1 {
					arg = args[1]
				}
				if arg == "" {
					ctx.Reply(T().ShhOffUsage)
					return nil
				}
				phone, lid := ResolveTarget(ctx, arg)
				if phone == "" && lid == "" {
					ctx.Reply(T().UserResolveFail)
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

			// Normal: shh user
			phone, lid := ResolveTarget(ctx, args[0])
			if phone == "" && lid == "" {
				ctx.Reply(T().UserResolveFail)
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
