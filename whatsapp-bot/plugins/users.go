package plugins

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// LIDResolver is the subset of the lid_map store needed by the plugin system.
type LIDResolver interface {
	GetLIDForPN(ctx context.Context, pn types.JID) (types.JID, error)
	GetPNForLID(ctx context.Context, lid types.JID) (types.JID, error)
	PutLIDMapping(ctx context.Context, lid, pn types.JID) error
}

var lidResolver LIDResolver
var ownerPhone string // bare phone number of the bot owner (e.g. "2348062795602")

// InitLIDStore wires the LID↔PN resolver and records the owner's phone number.
// ownerPN should be the bare phone number extracted from client.Store.ID.
func InitLIDStore(ls LIDResolver, ownerPN string) {
	lidResolver = ls
	ownerPhone = ownerPN
}

// GetAltID returns the alternate identifier for the given user string.
// Input may be:
//   - a bare phone number ("2348062795602") → returns the LID user part
//   - a full JID string ("2348062795602@s.whatsapp.net") → returns LID user part
//   - a LID user part or full LID JID ("270613692313713@lid") → returns phone number
//
// Returns an empty string when no mapping is found.
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
		// Bare string: treat as phone number.
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

// SaveUser persists LID↔PN mappings extracted from the message event.
//
// Cases handled:
//  1. Any incoming message: SenderAlt carries the sender's phone directly.
//  2. Our own outgoing message: ownerPhone is our PN, Sender is our LID.
//  3. Our own outgoing DM: Chat is recipient's LID, RecipientAlt is their phone.
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

	// Case 1: WhatsApp provides the sender's phone in SenderAlt for all
	// incoming messages (DM and group). This is the most reliable source.
	if evt.Info.SenderAlt.User != "" && evt.Info.SenderAlt.Server == types.DefaultUserServer {
		pnJID := types.NewJID(evt.Info.SenderAlt.User, types.DefaultUserServer)
		_ = lidResolver.PutLIDMapping(ctx, senderLID, pnJID)
	} else if evt.Info.IsFromMe && ownerPhone != "" {
		// Case 2: Our own outgoing message — sender LID is ours.
		pnJID := types.NewJID(ownerPhone, types.DefaultUserServer)
		_ = lidResolver.PutLIDMapping(ctx, senderLID, pnJID)
	}

	// Case 3: Outgoing DM — Chat is the recipient's LID, RecipientAlt is their phone.
	if evt.Info.IsFromMe && !evt.Info.IsGroup &&
		evt.Info.Chat.Server == types.HiddenUserServer &&
		evt.Info.RecipientAlt.User != "" && evt.Info.RecipientAlt.Server == types.DefaultUserServer {
		recipLID := types.NewJID(evt.Info.Chat.User, types.HiddenUserServer)
		recipPN := types.NewJID(evt.Info.RecipientAlt.User, types.DefaultUserServer)
		_ = lidResolver.PutLIDMapping(ctx, recipLID, recipPN)
	}
}

// BootstrapOwnerSudoers adds the owner's phone (and LID if already resolvable)
// to the sudoers list and persists the change. Safe to call multiple times.
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

// ResolveTarget determines the target user for commands like setsudo.
// Resolution order:
//  1. Reply — if arg is empty or "reply", use contextInfo.participant (quoted sender's LID).
//  2. Mention — if arg starts with "@", strip it and treat as phone.
//  3. Explicit — arg is a bare phone number or LID string.
//
// Returns (phone, lid); either may be empty when the store has no mapping yet.
func ResolveTarget(ctx *Context, arg string) (phone, lid string) {
	// 1. Reply: get the quoted message sender from contextInfo.
	if arg == "" || strings.EqualFold(arg, "reply") {
		participant := ctx.Event.Message.GetExtendedTextMessage().GetContextInfo().GetParticipant()
		if participant != "" {
			return resolveJIDString(participant)
		}
		if arg != "" {
			// "reply" was explicit but there is no quoted message.
			return "", ""
		}
	}

	// 2. @mention: strip "@" and treat remainder as phone.
	arg = strings.TrimPrefix(arg, "@")

	// 3. Explicit phone or LID.
	return resolveJIDString(arg)
}

// resolveJIDString takes a JID string, bare phone number, or LID user part
// and returns (phone, lid) by consulting the LID store for whichever side is
// missing.
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
		// Drop device suffix (e.g. "270613692313713:10@lid" → "270613692313713@lid").
		parsed.Device = 0
		jid = parsed
	} else {
		// Bare string: could be a phone number or a LID user part.
		// Heuristic: LID numbers tend to be longer (15 digits); phone numbers
		// are 7–15 digits. We can't reliably distinguish, so try phone first
		// (most common user input) then fall back to LID lookup.
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
