package plugins

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// isGroupStatusMsg returns true for messages that are group status updates
// (status mentions, group status posts) that antistatus should remove.
func isGroupStatusMsg(evt *events.Message) bool {
	m := evt.Message
	if m.GetGroupStatusMentionMessage() != nil {
		return true
	}
	if m.GetGroupStatusMessage() != nil {
		return true
	}
	if m.GetGroupStatusMessageV2() != nil {
		return true
	}
	// Also catch via ContextInfo flag.
	if ci := m.GetExtendedTextMessage().GetContextInfo(); ci != nil && ci.GetIsGroupStatus() {
		return true
	}
	if pm := m.GetProtocolMessage(); pm != nil {
		if pm.GetType() == waProto.ProtocolMessage_STATUS_MENTION_MESSAGE {
			return true
		}
	}
	if m.GetGroupInviteMessage() != nil {
		return true
	}
	return false
}

func revokeMsg(client *whatsmeow.Client, chat, sender types.JID, msgID string) {
	revoke := client.BuildRevoke(chat, sender, types.MessageID(msgID))
	client.SendMessage(context.Background(), chat, revoke)
}

func menuHeader(name string) string {
	return fmt.Sprintf("*.%s*\n-------------------\n", name)
}

func sendMentionToChat(client *whatsmeow.Client, chat types.JID, text string, jids []string) {
	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: jids,
			},
		},
	}
	client.SendMessage(context.Background(), chat, msg)
}

func relativeDuration(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%.0fs ago", d.Seconds())
	case d < time.Hour:
		return fmt.Sprintf("%.0fm ago", d.Minutes())
	case d < 24*time.Hour:
		return fmt.Sprintf("%.0fh ago", d.Hours())
	default:
		return fmt.Sprintf("%.0fd ago", d.Hours()/24)
	}
}

// ── antispam state ────────────────────────────────────────────────────────────

var (
	spamTimestamps = map[string][]time.Time{}
	spamMsgIDs     = map[string][]string{}
	spamMu         sync.Mutex

	dmSpamTimestamps = map[string][]time.Time{}
	dmSpamWarned     = map[string]bool{}
	dmSpamMu         sync.Mutex

	afkCooldown   = map[string]time.Time{}
	afkCooldownMu sync.Mutex
)

// ── hook registration ─────────────────────────────────────────────────────────

func init() {
	RegisterModerationHook(moderationHook)
}

func moderationHook(client *whatsmeow.Client, evt *events.Message) {
	// Derive our own phone and LID from the client store.
	myPhone := ""
	myLID := ""
	if client.Store.ID != nil {
		myPhone = strings.SplitN(client.Store.ID.User, ".", 2)[0]
	}
	myLID = client.Store.LID.User

	// Recompute isFromMe: compare sender/senderAlt against our phone and LID.
	// Guard against empty-string false positives (only compare when non-empty).
	su := evt.Info.Sender.User
	sa := evt.Info.SenderAlt.User
	isFromMe := false
	if myPhone != "" && (su == myPhone || sa == myPhone) {
		isFromMe = true
	}
	if !isFromMe && myLID != "" && (su == myLID || sa == myLID) {
		isFromMe = true
	}
	// Fall back to whatsmeow's own flag when we can't determine from store.
	if myPhone == "" && myLID == "" {
		isFromMe = evt.Info.IsFromMe
	}

	// When the owner sends any message, clear their AFK —
	// but not when they're explicitly setting it.
	if isFromMe {
		if ownerPhone != "" {
			text := extractMsgText(evt)
			_, name, _, ok := parseCommand(text, BotSettings.GetPrefixes())
			if !ok || name != "afk" {
				clearAFK(ownerPhone)
			}
		}
		return
	}

	chatJID := evt.Info.Chat.String()
	senderUser := evt.Info.Sender.User
	senderAlt := evt.Info.SenderAlt.User
	isGroup := evt.Info.Chat.Server == types.GroupServer
	msgText := extractMsgText(evt)

	// ── AFK auto-reply ────────────────────────────────────────────────────────
	if ownerPhone != "" {
		if status := getAFK(ownerPhone); status != nil {
			shouldReply := false
			if !isGroup {
				shouldReply = true
			} else {
				participant := evt.Message.GetExtendedTextMessage().GetContextInfo().GetParticipant()
				mentionedJIDs := evt.Message.GetExtendedTextMessage().GetContextInfo().GetMentionedJID()
				if participant != "" {
					partUser := strings.Split(participant, "@")[0]
					if partUser == ownerPhone || (myLID != "" && partUser == myLID) {
						shouldReply = true
					}
				}
				if !shouldReply {
					for _, jid := range mentionedJIDs {
						if strings.HasPrefix(jid, ownerPhone+"@") || (myLID != "" && strings.HasPrefix(jid, myLID+"@")) {
							shouldReply = true
							break
						}
					}
				}
			}
			if shouldReply {
				cooldownKey := chatJID + ":" + senderUser
				afkCooldownMu.Lock()
				lastSent, ok := afkCooldown[cooldownKey]
				elapsed := time.Since(lastSent)
				if !ok || elapsed >= 30*time.Second {
					afkCooldown[cooldownKey] = time.Now()
					afkCooldownMu.Unlock()
					lastSeen := status.SetAt.Format("3:04PM, 02 Jan 2006")
					reply := fmt.Sprintf(T().AFKAutoReply, lastSeen)
					if status.Message != "" {
						reply += "\n\n" + status.Message
					}
					reply += "\n\n" + T().AFKDefaultMsg
					client.SendMessage(context.Background(), evt.Info.Chat,
						&waProto.Message{Conversation: proto.String(reply)})
				} else {
					afkCooldownMu.Unlock()
				}
			}
		}
	}

	// ── DM antispam ───────────────────────────────────────────────────────────
	if !isGroup {
		// Skip old messages arriving during device sync.
		if time.Since(evt.Info.Timestamp) > 30*time.Second {
			return
		}
		dmKey := senderUser
		dmSpamMu.Lock()
		now := time.Now()
		dmSpamTimestamps[dmKey] = append(dmSpamTimestamps[dmKey], now)
		var recent []time.Time
		for _, t := range dmSpamTimestamps[dmKey] {
			if now.Sub(t) <= 5*time.Second {
				recent = append(recent, t)
			}
		}
		dmSpamTimestamps[dmKey] = recent
		count := len(recent)
		warned := dmSpamWarned[dmKey]
		dmSpamMu.Unlock()

		if count > 3 {
			if warned {
				senderJID := types.NewJID(senderUser, types.DefaultUserServer)
				if senderAlt != "" {
					senderJID = types.NewJID(senderAlt, types.DefaultUserServer)
				}
				client.UpdateBlocklist(context.Background(), senderJID, events.BlocklistChangeActionBlock)
				dmSpamMu.Lock()
				delete(dmSpamWarned, dmKey)
				delete(dmSpamTimestamps, dmKey)
				dmSpamMu.Unlock()
			} else {
				client.SendMessage(context.Background(), evt.Info.Chat,
					&waProto.Message{Conversation: proto.String(T().AntispamWarn)})
				dmSpamMu.Lock()
				dmSpamWarned[dmKey] = true
				dmSpamMu.Unlock()
			}
		}
		return
	}

	// ── Group-specific moderation ─────────────────────────────────────────────
	var (
		participants    []types.GroupParticipant
		groupInfoLoaded bool
	)
	botJID := client.Store.ID.ToNonAD()

	loadGroup := func() {
		if !groupInfoLoaded {
			groupInfoLoaded = true
			if gi, err := client.GetGroupInfo(context.Background(), evt.Info.Chat); err == nil {
				participants = gi.Participants
			}
		}
	}

	isBotAdmin := func() bool {
		loadGroup()
		return botIsAdmin(participants, ownerPhone, botJID.User)
	}

	isSenderAdmin := func() bool {
		loadGroup()
		p := findParticipant(participants, senderUser, "")
		if p == nil && senderAlt != "" {
			p = findParticipant(participants, senderAlt, "")
		}
		return p != nil && (p.IsAdmin || p.IsSuperAdmin)
	}

	// 0. antistatus
	if getAntistatusEnabled(chatJID) && isBotAdmin() && !isSenderAdmin() {
		if isGroupStatusMsg(evt) {
			revokeMsg(client, evt.Info.Chat, evt.Info.Sender, string(evt.Info.ID))
			senderJIDStr := evt.Info.Sender.ToNonAD().String()
			notify := fmt.Sprintf(T().AntistatusNotify, senderUser)
			sendMentionToChat(client, evt.Info.Chat, notify, []string{senderJIDStr})
			return
		}
	}

	// 1. shh
	if isShhed(chatJID, senderUser) && isBotAdmin() {
		revokeMsg(client, evt.Info.Chat, evt.Info.Sender, string(evt.Info.ID))
		return
	}

	// 2. antilink
	if mode := getAntilinkMode(chatJID); mode != "off" && msgText != "" {
		if isBotAdmin() && !isSenderAdmin() {
			if urlRegex.MatchString(msgText) {
				revokeMsg(client, evt.Info.Chat, evt.Info.Sender, string(evt.Info.ID))
				senderJIDStr := evt.Info.Sender.ToNonAD().String()
				notify := fmt.Sprintf(T().AntilinkNotify, senderUser)
				sendMentionToChat(client, evt.Info.Chat, notify, []string{senderJIDStr})
				if mode == "kick" {
					client.UpdateGroupParticipants(context.Background(), evt.Info.Chat,
						[]types.JID{evt.Info.Sender.ToNonAD()}, whatsmeow.ParticipantChangeRemove)
				}
				return
			}
		}
	}

	// 3. antiword
	if words := getAntiwords(chatJID); len(words) > 0 && msgText != "" {
		if isBotAdmin() && !isSenderAdmin() {
			lower := strings.ToLower(msgText)
			for _, w := range words {
				if strings.Contains(lower, strings.ToLower(w)) {
					revokeMsg(client, evt.Info.Chat, evt.Info.Sender, string(evt.Info.ID))
					return
				}
			}
		}
	}

	// 4. antispam (group)
	if getAntispamMode(chatJID) != "off" {
		// Skip old messages arriving during device sync.
		if time.Since(evt.Info.Timestamp) <= 30*time.Second && !isAntispamWhitelisted(chatJID, senderUser) {
			spamKey := chatJID + ":" + senderUser
			spamMu.Lock()
			now := time.Now()
			spamTimestamps[spamKey] = append(spamTimestamps[spamKey], now)
			spamMsgIDs[spamKey] = append(spamMsgIDs[spamKey], string(evt.Info.ID))
			var recentTimes []time.Time
			var recentIDs []string
			for i, t := range spamTimestamps[spamKey] {
				if now.Sub(t) <= 5*time.Second {
					recentTimes = append(recentTimes, t)
					recentIDs = append(recentIDs, spamMsgIDs[spamKey][i])
				}
			}
			spamTimestamps[spamKey] = recentTimes
			spamMsgIDs[spamKey] = recentIDs
			count := len(recentTimes)
			allIDs := make([]string, len(recentIDs))
			copy(allIDs, recentIDs)
			spamMu.Unlock()

			if count > 3 && isBotAdmin() {
				for _, msgID := range allIDs {
					revokeMsg(client, evt.Info.Chat, evt.Info.Sender, msgID)
				}
				client.UpdateGroupParticipants(context.Background(), evt.Info.Chat,
					[]types.JID{evt.Info.Sender.ToNonAD()}, whatsmeow.ParticipantChangeRemove)
			}
		}
	}
}
