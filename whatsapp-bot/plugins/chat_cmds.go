package plugins

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow/appstate"
	waCommon "go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

func init() {
	Register(&Command{
		Pattern:  "star",
		Category: "utility",
		Func:     starCmd,
	})
	Register(&Command{
		Pattern:  "unstar",
		Category: "utility",
		Func:     unstarCmd,
	})
	Register(&Command{
		Pattern:  "pin",
		Category: "utility",
		Func:     pinCmd,
	})
	Register(&Command{
		Pattern:  "unpin",
		Category: "utility",
		Func:     unpinCmd,
	})
	Register(&Command{
		Pattern:  "archive",
		Category: "utility",
		Func:     archiveCmd,
	})
	Register(&Command{
		Pattern:  "unarchive",
		Category: "utility",
		Func:     unarchiveCmd,
	})
	Register(&Command{
		Pattern:  "leave",
		Category: "group",
		IsGroup:  true,
		IsAdmin:  true,
		Func:     leaveCmd,
	})
	Register(&Command{
		Pattern:  "clear",
		Category: "utility",
		Func:     clearCmd,
	})
}

func pinCmd(ctx *Context) error {
	return pinToggle(ctx, true)
}

func unpinCmd(ctx *Context) error {
	return pinToggle(ctx, false)
}

func pinToggle(ctx *Context, pin bool) error {
	chat := ctx.Event.Info.Chat
	rawMsg := ctx.Event.Message

	ci := rawMsg.GetExtendedTextMessage().GetContextInfo()
	msgID := ci.GetStanzaID()
	participant := ci.GetParticipant()

	// If replying to a message, pin/unpin that specific message.
	if msgID != "" {
		var senderJID types.JID
		if participant != "" {
			if parsed, err := types.ParseJID(participant); err == nil {
				senderJID = parsed.ToNonAD()
			} else {
				senderJID = ctx.Event.Info.Sender.ToNonAD()
			}
		} else {
			senderJID = ctx.Event.Info.Sender.ToNonAD()
		}

		isFromMe := isOwnJID(ctx.Client, senderJID.User)
		key := &waCommon.MessageKey{
			RemoteJID: proto.String(chat.String()),
			FromMe:    proto.Bool(isFromMe),
			ID:        proto.String(msgID),
		}
		if chat.Server == types.GroupServer {
			s := senderJID.String()
			key.Participant = proto.String(s)
		}

		pinType := waE2E.PinInChatMessage_PIN_FOR_ALL
		var duration uint32 = 604800 // 7 days
		if !pin {
			pinType = waE2E.PinInChatMessage_UNPIN_FOR_ALL
			duration = 0
		}

		msg := &waE2E.Message{
			MessageContextInfo: &waE2E.MessageContextInfo{
				MessageAddOnDurationInSecs: proto.Uint32(duration),
			},
			PinInChatMessage: &waE2E.PinInChatMessage{
				Key:               key,
				Type:              pinType.Enum(),
				SenderTimestampMS: proto.Int64(time.Now().UnixMilli()),
			},
		}

		if _, err := ctx.Client.SendMessage(context.Background(), chat, msg); err != nil {
			if pin {
				ctx.Reply(T().MsgPinFailed)
			} else {
				ctx.Reply(T().MsgUnpinFailed)
			}
			return nil
		}
		if pin {
			ctx.Reply(T().MsgPinOK)
		} else {
			ctx.Reply(T().MsgUnpinOK)
		}
		return nil
	}

	// No reply — pin/unpin the chat itself.
	patch := appstate.BuildPin(chat, pin)
	if err := ctx.Client.SendAppState(context.Background(), patch); err != nil {
		if pin {
			ctx.Reply(T().PinFailed)
		} else {
			ctx.Reply(T().UnpinFailed)
		}
		return nil
	}
	if pin {
		ctx.Reply(T().PinOK)
	} else {
		ctx.Reply(T().UnpinOK)
	}
	return nil
}

func archiveCmd(ctx *Context) error {
	patch := appstate.BuildArchive(ctx.Event.Info.Chat, true, ctx.Event.Info.Timestamp, nil)
	if err := ctx.Client.SendAppState(context.Background(), patch); err != nil {
		ctx.Reply(T().ArchiveFailed)
		return nil
	}
	ctx.Reply(T().ArchiveOK)
	return nil
}

func unarchiveCmd(ctx *Context) error {
	patch := appstate.BuildArchive(ctx.Event.Info.Chat, false, ctx.Event.Info.Timestamp, nil)
	if err := ctx.Client.SendAppState(context.Background(), patch); err != nil {
		ctx.Reply(T().UnarchiveFailed)
		return nil
	}
	ctx.Reply(T().UnarchiveOK)
	return nil
}

func leaveCmd(ctx *Context) error {
	ctx.Reply(T().LeaveOK)
	return ctx.Client.LeaveGroup(context.Background(), ctx.Event.Info.Chat)
}

func clearCmd(ctx *Context) error {
	chat := ctx.Event.Info.Chat
	patch := appstate.BuildClearChat(chat, ctx.Event.Info.Timestamp, nil)
	if err := ctx.Client.SendAppState(context.Background(), patch); err != nil {
		ctx.Reply(T().ClearFailed)
		return nil
	}
	ctx.Reply(T().ClearOK)
	return nil
}

func starCmd(ctx *Context) error {
	return starToggle(ctx, true)
}

func unstarCmd(ctx *Context) error {
	return starToggle(ctx, false)
}

func starToggle(ctx *Context, starred bool) error {
	ci := ctx.Event.Message.GetExtendedTextMessage().GetContextInfo()
	msgID := types.MessageID(ci.GetStanzaID())
	participant := ci.GetParticipant()
	if msgID == "" {
		if starred {
			ctx.Reply(T().StarUsage)
		} else {
			ctx.Reply(T().UnstarUsage)
		}
		return nil
	}

	chat := ctx.Event.Info.Chat
	var senderJID types.JID
	if participant != "" {
		if parsed, err := types.ParseJID(participant); err == nil {
			senderJID = parsed.ToNonAD()
		}
	} else {
		senderJID = ctx.Event.Info.Sender.ToNonAD()
	}
	fromMe := isOwnJID(ctx.Client, senderJID.User)

	patch := appstate.BuildStar(chat, senderJID, msgID, fromMe, starred)
	if err := ctx.Client.SendAppState(context.Background(), patch); err != nil {
		if starred {
			ctx.Reply(T().StarFailed)
		} else {
			ctx.Reply(T().UnstarFailed)
		}
		return nil
	}
	if starred {
		ctx.Reply(T().StarOK)
	} else {
		ctx.Reply(T().UnstarOK)
	}
	return nil
}
