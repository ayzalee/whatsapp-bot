package plugins

import (
	"context"
	"sync"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

var MetaJID = types.NewMetaAIJID

var metaMu sync.Mutex

var pendingReplies = make(map[string]types.JID)

var lastProcessedResponse = make(map[string]string)

var sentMessageIDs = make(map[string]types.MessageID)

func HandleMetaAIResponse(client *whatsmeow.Client, v *events.Message) {
	var responseText string
	resID := v.Message.GetMessageContextInfo().GetBotMetadata().GetBotResponseID()

	if img := v.Message.GetImageMessage(); img != nil {
		metaMu.Lock()
		targetJID, ok := pendingReplies[v.Info.Sender.String()]
		metaMu.Unlock()
		if ok {
			client.SendMessage(context.Background(), targetJID, &waProto.Message{ImageMessage: img})
		}
		return
	}

	if v.Message.Conversation != nil {
		responseText = v.Message.GetConversation()
	} else if v.Message.ExtendedTextMessage != nil {
		responseText = v.Message.GetExtendedTextMessage().GetText()
	} else if v.Message.ProtocolMessage != nil &&
		v.Message.ProtocolMessage.GetType() == waProto.ProtocolMessage_MESSAGE_EDIT {
		edit := v.Message.ProtocolMessage.EditedMessage
		if edit != nil {
			if edit.Conversation != nil {
				responseText = edit.GetConversation()
			} else if edit.ExtendedTextMessage != nil {
				responseText = edit.ExtendedTextMessage.GetText()
			}
		}
	}

	if responseText == "" || resID == "" {
		return
	}

	metaMu.Lock()
	defer metaMu.Unlock()

	if lastText, seen := lastProcessedResponse[resID]; seen && len(responseText) <= len(lastText) {
		return
	}

	targetJID, ok := pendingReplies[v.Info.Sender.String()]
	if !ok {
		return
	}

	if msgID, exists := sentMessageIDs[resID]; exists {
		editMsg := client.BuildEdit(targetJID, msgID, &waProto.Message{
			Conversation: proto.String(responseText),
		})
		if _, err := client.SendMessage(context.Background(), targetJID, editMsg); err == nil {
			lastProcessedResponse[resID] = responseText
		}
	} else {
		if resp, err := client.SendMessage(context.Background(), targetJID, &waProto.Message{
			Conversation: proto.String(responseText),
		}); err == nil {
			sentMessageIDs[resID] = resp.ID
			lastProcessedResponse[resID] = responseText
		}
	}
}

func init() {
	Register(&Command{
		Pattern:  "meta",
		Category: "ai",
		Func: func(ctx *Context) error {
			query := ctx.Text

			var outMsg *waProto.Message

			if img := ctx.Event.Message.GetImageMessage(); img != nil {
				img.Caption = proto.String(query)
				outMsg = &waProto.Message{ImageMessage: img}
			} else if vid := ctx.Event.Message.GetVideoMessage(); vid != nil {
				vid.Caption = proto.String(query)
				outMsg = &waProto.Message{VideoMessage: vid}
			} else if ext := ctx.Event.Message.GetExtendedTextMessage(); ext != nil {

				quoted := ext.GetContextInfo().GetQuotedMessage()
				if quoted.GetImageMessage() != nil || quoted.GetVideoMessage() != nil {
					if query == "" {
						ctx.Reply(T().MetaUsage)
						return nil
					}
					ext.Text = proto.String(query)
					outMsg = &waProto.Message{ExtendedTextMessage: ext}
				}
			}

			if outMsg == nil {
				if query == "" {
					ctx.Reply(T().MetaUsage)
					return nil
				}
				outMsg = &waProto.Message{Conversation: proto.String(query)}
			}

			resp, err := ctx.Client.SendMessage(context.Background(), MetaJID, outMsg)
			if err != nil {
				return err
			}
			_ = resp

			metaMu.Lock()
			pendingReplies[MetaJID.String()] = ctx.Event.Info.Chat
			metaMu.Unlock()
			return nil
		},
	})
}
