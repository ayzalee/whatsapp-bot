package plugins

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

var antichEnabled = make(map[string]bool)

func isChannelForwardMsg(evt *events.Message) bool {
	msg := evt.Message

	if msg.GetNewsletterAdminInviteMessage() != nil {
		return true
	}

	if msg.GetTemplateMessage() != nil {
		return true
	}

	extText := msg.GetExtendedTextMessage()
	if extText == nil {
		if conv := msg.GetConversation(); conv != "" {
			return strings.Contains(conv, "whatsapp.com/channel")
		}
		return false
	}

	text := extText.GetText()
	matchedText := extText.GetMatchedText()

	if strings.Contains(text, "whatsapp.com/channel") ||
		strings.Contains(matchedText, "whatsapp.com/channel") {
		return true
	}

	ci := extText.GetContextInfo()
	if ci == nil {
		return false
	}

	isForwarded := ci.GetIsForwarded()
	forwardScore := ci.GetForwardingScore()

	if isForwarded && forwardScore >= 100 && len(text) > 0 && len(text) < 100 {
		return true
	}

	return false
}

func init() {
	RegisterModerationHook(func(client *whatsmeow.Client, evt *events.Message) {
		if !evt.Info.IsGroup {
			return
		}
		chatJID := evt.Info.Chat.String()
		if !antichEnabled[chatJID] {
			return
		}
		if isChannelForwardMsg(evt) {
			client.SendMessage(context.Background(), evt.Info.Chat,
				client.BuildRevoke(evt.Info.Chat, evt.Info.Sender, evt.Info.ID))
		}
	})

	Register(&Command{
		Pattern:  "antich",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			chatJID := ctx.Event.Info.Chat.String()
			arg := strings.ToLower(strings.TrimSpace(ctx.Text))
			switch arg {
			case "on":
				antichEnabled[chatJID] = true
				ctx.Reply("Anti-channel forward enabled.")
			case "off":
				antichEnabled[chatJID] = false
				ctx.Reply("Anti-channel forward disabled.")
			default:
				status := "off"
				if antichEnabled[chatJID] {
					status = "on"
				}
				ctx.Reply("*Anti Channel Forward*\nStatus: " + status + "\n\n.antich on\n.antich off")
			}
			return nil
		},
	})
}
