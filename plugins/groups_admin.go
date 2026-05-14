package plugins

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// findParticipant finds a group participant by phone number or LID.
func findParticipant(participants []types.GroupParticipant, phone, lid string) *types.GroupParticipant {
	for i := range participants {
		p := &participants[i]
		if phone != "" && (p.JID.User == phone || p.PhoneNumber.User == phone) {
			return p
		}
		if lid != "" && (p.JID.User == lid || p.LID.User == lid) {
			return p
		}
	}
	return nil
}

// botIsAdmin reports whether the bot (identified by phone or user) is an admin.
func botIsAdmin(participants []types.GroupParticipant, phone, user string) bool {
	p := findParticipant(participants, phone, user)
	if p == nil {
		return false
	}
	return p.IsAdmin || p.IsSuperAdmin
}

func init() {
	Register(&Command{
		Pattern:  "promote",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			arg := ctx.Text
			if arg == "" {
				ctx.Reply(menuHeader("promote") + T().PromoteUsage)
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
			if p.IsAdmin || p.IsSuperAdmin {
				ctx.Reply(T().PromoteAlreadyAdmin)
				return nil
			}
			targetJID := p.JID.ToNonAD()
			_, err = ctx.Client.UpdateGroupParticipants(context.Background(), ctx.Event.Info.Chat,
				[]types.JID{targetJID}, whatsmeow.ParticipantChangePromote)
			if err != nil {
				return err
			}
			display := phone
			if display == "" {
				display = lid
			}
			ctx.Reply(fmt.Sprintf(T().PromoteOK, display))
			return nil
		},
	})

	Register(&Command{
		Pattern:  "demote",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			arg := ctx.Text
			if arg == "" {
				ctx.Reply(menuHeader("demote") + T().DemoteUsage)
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
			if p.IsSuperAdmin {
				ctx.Reply(T().DemoteSuperAdmin)
				return nil
			}
			if !p.IsAdmin {
				ctx.Reply(T().DemoteNotAdmin)
				return nil
			}
			targetJID := p.JID.ToNonAD()
			_, err = ctx.Client.UpdateGroupParticipants(context.Background(), ctx.Event.Info.Chat,
				[]types.JID{targetJID}, whatsmeow.ParticipantChangeDemote)
			if err != nil {
				return err
			}
			display := phone
			if display == "" {
				display = lid
			}
			ctx.Reply(fmt.Sprintf(T().DemoteOK, display))
			return nil
		},
	})

	Register(&Command{
		Pattern:  "kick",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			arg := ctx.Text
			if arg == "" {
				ctx.Reply(menuHeader("kick") + T().KickUsage)
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
			if p.IsSuperAdmin {
				ctx.Reply(T().KickSuperAdmin)
				return nil
			}
			targetJID := p.JID.ToNonAD()
			_, err = ctx.Client.UpdateGroupParticipants(context.Background(), ctx.Event.Info.Chat,
				[]types.JID{targetJID}, whatsmeow.ParticipantChangeRemove)
			if err != nil {
				return err
			}
			display := phone
			if display == "" {
				display = lid
			}
			ctx.Reply(fmt.Sprintf(T().KickOK, display))
			return nil
		},
	})

	Register(&Command{
		Pattern:  "kickall",
		IsGroup:  true,
		IsAdmin:  true,
		IsSudo:   true,
		Category: "group",
		Func: func(ctx *Context) error {
			group, err := ctx.Client.GetGroupInfo(context.Background(), ctx.Event.Info.Chat)
			if err != nil {
				ctx.Reply(fmt.Sprintf(T().GroupInfoFailed, err.Error()))
				return nil
			}
			var toKick []types.JID
			for _, p := range group.Participants {
				if !p.IsSuperAdmin {
					toKick = append(toKick, p.JID.ToNonAD())
				}
			}
			ctx.Reply(fmt.Sprintf(T().KickAllStart, len(toKick)))
			for i := 0; i < len(toKick); i += 20 {
				end := i + 20
				if end > len(toKick) {
					end = len(toKick)
				}
				ctx.Client.UpdateGroupParticipants(context.Background(), ctx.Event.Info.Chat,
					toKick[i:end], whatsmeow.ParticipantChangeRemove)
			}
			ctx.Reply(T().KickAllDone)
			ctx.Client.LeaveGroup(context.Background(), ctx.Event.Info.Chat)
			return nil
		},
	})

	Register(&Command{
		Pattern:  "mute",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			group, err := ctx.Client.GetGroupInfo(context.Background(), ctx.Event.Info.Chat)
			if err != nil {
				ctx.Reply(fmt.Sprintf(T().GroupInfoFailed, err.Error()))
				return nil
			}
			if group.IsAnnounce {
				ctx.Reply(T().MuteAlready)
				return nil
			}
			if err := ctx.Client.SetGroupAnnounce(context.Background(), ctx.Event.Info.Chat, true); err != nil {
				return err
			}
			ctx.Reply(T().MuteOK)
			return nil
		},
	})

	Register(&Command{
		Pattern:  "unmute",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			group, err := ctx.Client.GetGroupInfo(context.Background(), ctx.Event.Info.Chat)
			if err != nil {
				ctx.Reply(fmt.Sprintf(T().GroupInfoFailed, err.Error()))
				return nil
			}
			if !group.IsAnnounce {
				ctx.Reply(T().UnmuteNotMuted)
				return nil
			}
			if err := ctx.Client.SetGroupAnnounce(context.Background(), ctx.Event.Info.Chat, false); err != nil {
				return err
			}
			ctx.Reply(T().UnmuteOK)
			return nil
		},
	})

	Register(&Command{
		Pattern:  "messages",
		Category: "group",
		Func: func(ctx *Context) error {
			rows, err := settingsDB.Query(
				`SELECT chat_jid, COUNT(*) as cnt FROM message_secrets WHERE chat_jid != 'status@broadcast' GROUP BY chat_jid ORDER BY cnt DESC LIMIT 30`,
			)
			if err != nil {
				return err
			}
			defer rows.Close()

			var sb strings.Builder
			sb.WriteString(T().MessagesHeader)
			n := 0
			for rows.Next() {
				var jidStr string
				var cnt int
				if err := rows.Scan(&jidStr, &cnt); err != nil {
					continue
				}
				if strings.HasSuffix(jidStr, "@bot") {
					continue
				}
				var name string
				if strings.HasSuffix(jidStr, "@g.us") {
					parsed, err := types.ParseJID(jidStr)
					if err == nil {
						if gi, err := ctx.Client.GetGroupInfo(context.Background(), parsed); err == nil {
							name = gi.Name
						}
					}
				} else if strings.HasSuffix(jidStr, "@s.whatsapp.net") {
					var pushName string
					settingsDB.QueryRow(`SELECT push_name FROM contacts WHERE their_jid = ?`, jidStr).Scan(&pushName)
					if pushName != "" {
						name = pushName
					}
				} else if strings.HasSuffix(jidStr, "@lid") {
					userPart := strings.TrimSuffix(jidStr, "@lid")
					var pushName string
					settingsDB.QueryRow(
						`SELECT c.push_name FROM lid_map l JOIN contacts c ON c.their_jid = l.pn || '@s.whatsapp.net' WHERE l.lid = ?`,
						userPart,
					).Scan(&pushName)
					if pushName != "" {
						name = pushName
					}
				}
				if name == "" {
					continue
				}
				n++
				fmt.Fprintf(&sb, "%d. %s — %d msgs\n", n, name, cnt)
			}
			if n == 0 {
				ctx.Reply(T().MessagesEmpty)
				return nil
			}
			ctx.Reply(strings.TrimRight(sb.String(), "\n"))
			return nil
		},
	})

	Register(&Command{
		Pattern:  "active",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			chatJID := ctx.Event.Info.Chat.String()
			rows, err := settingsDB.Query(
				`SELECT sender_jid, COUNT(*) as cnt FROM message_secrets WHERE chat_jid = ? GROUP BY sender_jid ORDER BY cnt DESC LIMIT 20`,
				chatJID,
			)
			if err != nil {
				return err
			}
			defer rows.Close()

			var sb strings.Builder
			sb.WriteString(T().ActiveHeader)
			var mentions []string
			n := 0
			for rows.Next() {
				var senderJID string
				var cnt int
				if err := rows.Scan(&senderJID, &cnt); err != nil {
					continue
				}
				n++
				// senderJID is already a full JID string from the DB
				userPart := senderJID
				if idx := strings.Index(senderJID, "@"); idx != -1 {
					userPart = senderJID[:idx]
				}
				sb.WriteString(fmt.Sprintf("%d. @%s — %d msgs\n", n, userPart, cnt))
				mentions = append(mentions, senderJID)
			}
			if n == 0 {
				ctx.Reply(T().ActiveEmpty)
				return nil
			}
			sendMention(ctx, strings.TrimRight(sb.String(), "\n"), mentions)
			return nil
		},
	})

	Register(&Command{
		Pattern:  "inactive",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			group, err := ctx.Client.GetGroupInfo(context.Background(), ctx.Event.Info.Chat)
			if err != nil {
				ctx.Reply(fmt.Sprintf(T().GroupInfoFailed, err.Error()))
				return nil
			}

			chatJID := ctx.Event.Info.Chat.String()
			rows, err := settingsDB.Query(
				`SELECT sender_jid, COUNT(*) as cnt FROM message_secrets WHERE chat_jid = ? GROUP BY sender_jid`,
				chatJID,
			)
			if err != nil {
				return err
			}
			defer rows.Close()

			msgCounts := map[string]int{}
			for rows.Next() {
				var senderJID string
				var cnt int
				if rows.Scan(&senderJID, &cnt) == nil {
					userPart := senderJID
					if idx := strings.Index(senderJID, "@"); idx != -1 {
						userPart = senderJID[:idx]
					}
					msgCounts[userPart] = cnt
				}
			}

			// resolve message count for a participant, checking JID, LID and PhoneNumber
			getMsgCount := func(p types.GroupParticipant) int {
				if cnt, ok := msgCounts[p.JID.User]; ok {
					return cnt
				}
				if p.LID.User != "" {
					if cnt, ok := msgCounts[p.LID.User]; ok {
						return cnt
					}
				}
				if p.PhoneNumber.User != "" {
					if cnt, ok := msgCounts[p.PhoneNumber.User]; ok {
						return cnt
					}
				}
				return 0
			}

			// inactive = participants with zero messages in this group
			type entry struct {
				jid types.GroupParticipant
				cnt int
			}
			var inactive []entry
			for _, p := range group.Participants {
				cnt := getMsgCount(p)
				if cnt == 0 {
					inactive = append(inactive, entry{p, 0})
				}
			}

			if len(inactive) == 0 {
				ctx.Reply(T().InactiveEmpty)
				return nil
			}
			if len(inactive) > 20 {
				inactive = inactive[:20]
			}

			var sb strings.Builder
			sb.WriteString(T().InactiveHeader)
			var mentions []string
			for i, e := range inactive {
				// prefer phone number for display
				displayUser := e.jid.PhoneNumber.User
				if displayUser == "" {
					displayUser = e.jid.JID.User
				}
				fullJID := displayUser + "@s.whatsapp.net"
				sb.WriteString(fmt.Sprintf("%d. @%s — 0 msgs\n", i+1, displayUser))
				mentions = append(mentions, fullJID)
			}
			sendMention(ctx, strings.TrimRight(sb.String(), "\n"), mentions)
			return nil
		},
	})
}
