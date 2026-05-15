package plugins

import (
	"context"
	"strings"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/proto/waSyncAction"
	"go.mau.fi/whatsmeow/types"
)

func init() {
	Register(&Command{
		Pattern:  "del",
		Category: "group",
		Func:     delCmd,
	})
}

// isOwnJID reports whether userPart belongs to the bot owner.
func isOwnJID(client *whatsmeow.Client, userPart string) bool {
	if userPart == "" {
		return false
	}
	if client.Store.ID != nil {
		myPhone := strings.SplitN(client.Store.ID.User, ".", 2)[0]
		if userPart == myPhone {
			return true
		}
	}
	if client.Store.LID.User != "" && userPart == client.Store.LID.User {
		return true
	}
	return false
}

func delCmd(ctx *Context) error {
	ci := ctx.Event.Message.GetExtendedTextMessage().GetContextInfo()
	stanzaID := ci.GetStanzaID()
	participant := ci.GetParticipant()

	if stanzaID == "" {
		ctx.Reply(T().DelUsage)
		return nil
	}

	chat := ctx.Event.Info.Chat
	isGroup := chat.Server == types.GroupServer
	msgID := types.MessageID(stanzaID)

	var targetSender types.JID
	if participant != "" {
		if parsed, err := types.ParseJID(participant); err == nil {
			targetSender = parsed.ToNonAD()
		}
	}

	// Determine whether the quoted message was sent by the bot owner.
	targetIsOwn := isOwnJID(ctx.Client, targetSender.User)

	// Per docs:
	//   - Own messages:             BuildRevoke(chat, types.EmptyJID, msgID) → send to chat
	//   - Others' messages (admin): BuildRevoke(chat, senderJID,     msgID) → send to chat
	//   - Delete for me:            BuildRevoke(chat, senderJID,     msgID) → send to self
	if targetIsOwn {
		// Revoking our own message — works in any context.
		ctx.Client.SendMessage(context.Background(), chat,
			ctx.Client.BuildRevoke(chat, types.EmptyJID, msgID))
		return nil
	}

	if isGroup {
		botAdmin := false
		if gi, err := ctx.Client.GetGroupInfo(context.Background(), chat); err == nil {
			botAdmin = botIsAdmin(gi.Participants, ownerPhone, ctx.Client.Store.ID.ToNonAD().User)
		}
		if botAdmin {
			// Delete for everyone.
			ctx.Client.SendMessage(context.Background(), chat,
				ctx.Client.BuildRevoke(chat, targetSender, msgID))
		} else {
			// Not admin — delete for me only.
			deleteForMe(ctx, chat, msgID, targetIsOwn, 0)
		}
	} else {
		// DM from someone else → delete for me via app state.
		deleteForMe(ctx, chat, msgID, false, 0)
	}
	return nil
}

// deleteForMe sends a deleteMessageForMeAction app state patch so WhatsApp
// removes the message only on the owner's devices.
func deleteForMe(ctx *Context, chatJID types.JID, msgID types.MessageID, fromMe bool, timestampUnix int64) {
	fromMeStr := "0"
	if fromMe {
		fromMeStr = "1"
	}
	patch := appstate.PatchInfo{
		Type: appstate.WAPatchRegularHigh,
		Mutations: []appstate.MutationInfo{{
			Index:   []string{"deleteMessageForMe", chatJID.String(), string(msgID), fromMeStr, "0"},
			Version: 3,
			Value: &waSyncAction.SyncActionValue{
				DeleteMessageForMeAction: &waSyncAction.DeleteMessageForMeAction{
					DeleteMedia:      proto.Bool(false),
					MessageTimestamp: proto.Int64(timestampUnix),
				},
			},
		}},
	}
	ctx.Client.SendAppState(context.Background(), patch)
}
