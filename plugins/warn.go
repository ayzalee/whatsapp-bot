package plugins

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

func init() {
	Register(&Command{
		Pattern:  "warn",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			arg := ""
			reason := ""
			if len(ctx.Args) > 0 {
				arg = ctx.Args[0]
				if len(ctx.Args) > 1 {
					reason = " " + strings.Join(ctx.Args[1:], " ")
				}
			}
			if arg == "" {
				ctx.Reply(T().WarnUsage)
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

			chatJID := ctx.Event.Info.Chat.String()
			userID := p.JID.User
			count := addWarn(chatJID, userID)

			reasonStr := ""
			if reason != "" {
				reasonStr = "\nReason:" + reason
			}
			senderJIDStr := p.JID.ToNonAD().String()
			warnMsg := fmt.Sprintf(T().WarnText, userID, reasonStr, count)
			sendMention(ctx, warnMsg, []string{senderJIDStr})

			if count >= 3 {
				ctx.Client.UpdateGroupParticipants(context.Background(), ctx.Event.Info.Chat,
					[]types.JID{p.JID.ToNonAD()}, whatsmeow.ParticipantChangeRemove)
				ctx.Reply(T().WarnKicked)
				resetWarns(chatJID, userID)
			}
			return nil
		},
	})
}
