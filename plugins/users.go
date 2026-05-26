package plugins

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type LIDResolver interface {
	GetLIDForPN(ctx context.Context, pn types.JID) (types.JID, error)
	GetPNForLID(ctx context.Context, lid types.JID) (types.JID, error)
	PutLIDMapping(ctx context.Context, lid, pn types.JID) error
}

var lidResolver LIDResolver
var ownerPhone string 

func InitLIDStore(ls LIDResolver, ownerPN string) {
	lidResolver = ls
	ownerPhone = ownerPN
}

func GetAltID(id string) string {
	if lidResolver == nil {
		return ""
	}
	ctx := context.Background()

	var jid types.JID
	if strings.Contains(id, "@") {
		parsed, err := types.ParseJID(id)
		if err != nil {
			return ""
		}
		jid = parsed
	} else {
		
		jid = types.NewJID(id, types.DefaultUserServer)
	}

	switch jid.Server {
	case types.DefaultUserServer:
		lid, err := lidResolver.GetLIDForPN(ctx, jid)
		if err != nil || lid.User == "" {
			return ""
		}
		return lid.User
	case types.HiddenUserServer:
		pn, err := lidResolver.GetPNForLID(ctx, jid)
		if err != nil || pn.User == "" {
			return ""
		}
		return pn.User
	}
	return ""
}

func SaveUser(evt *events.Message) {
	if lidResolver == nil {
		return
	}

	ctx := context.Background()
	sender := evt.Info.Sender

	if sender.Server != types.HiddenUserServer {
		return
	}
	senderLID := types.NewJID(sender.User, types.HiddenUserServer)

	
	
	if evt.Info.SenderAlt.User != "" && evt.Info.SenderAlt.Server == types.DefaultUserServer {
		pnJID := types.NewJID(evt.Info.SenderAlt.User, types.DefaultUserServer)
		_ = lidResolver.PutLIDMapping(ctx, senderLID, pnJID)
	} else if evt.Info.IsFromMe && ownerPhone != "" {
		
		pnJID := types.NewJID(ownerPhone, types.DefaultUserServer)
		_ = lidResolver.PutLIDMapping(ctx, senderLID, pnJID)
	}

	
	if evt.Info.IsFromMe && !evt.Info.IsGroup &&
		evt.Info.Chat.Server == types.HiddenUserServer &&
		evt.Info.RecipientAlt.User != "" && evt.Info.RecipientAlt.Server == types.DefaultUserServer {
		recipLID := types.NewJID(evt.Info.Chat.User, types.HiddenUserServer)
		recipPN := types.NewJID(evt.Info.RecipientAlt.User, types.DefaultUserServer)
		_ = lidResolver.PutLIDMapping(ctx, recipLID, recipPN)
	}
}

func BootstrapOwnerSudoers() {
	if ownerPhone == "" {
		return
	}
	changed := false

	if !BotSettings.IsSudo(ownerPhone) {
		BotSettings.AddSudo(ownerPhone)
		changed = true
	}

	if lid := GetAltID(ownerPhone); lid != "" && !BotSettings.IsSudo(lid) {
		BotSettings.AddSudo(lid)
		changed = true
	}

	if changed {
		_ = SaveSettings()
	}
}

func ResolveTarget(ctx *Context, arg string) (phone, lid string) {
	
	if arg == "" || strings.EqualFold(arg, "reply") {
		var participant string
		if ci := ctx.Event.Message.GetExtendedTextMessage().GetContextInfo(); ci != nil {
			participant = ci.GetParticipant()
		}
		if participant == "" {
			if ci := ctx.Event.Message.GetImageMessage().GetContextInfo(); ci != nil {
				participant = ci.GetParticipant()
			}
		}
		if participant == "" {
			if ci := ctx.Event.Message.GetVideoMessage().GetContextInfo(); ci != nil {
				participant = ci.GetParticipant()
			}
		}
		if participant != "" {
			return resolveJIDString(participant)
		}
		if arg != "" {
			return "", ""
		}
	}

	
	arg = strings.TrimPrefix(arg, "@")

	
	return resolveJIDString(arg)
}

func resolveJIDString(s string) (phone, lid string) {
	if s == "" {
		return "", ""
	}

	var jid types.JID
	if strings.Contains(s, "@") {
		parsed, err := types.ParseJID(s)
		if err != nil {
			return "", ""
		}
		
		parsed.Device = 0
		jid = parsed
	} else {
		
		
		
		
		s = strings.TrimPrefix(s, "+")
		jid = types.NewJID(s, types.DefaultUserServer)
	}

	switch jid.Server {
	case types.DefaultUserServer:
		phone = jid.User
		lid = GetAltID(jid.String())
	case types.HiddenUserServer:
		lid = jid.User
		phone = GetAltID(jid.String())
	}
	return
}
