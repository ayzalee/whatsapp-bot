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

// MetaJID is the WhatsApp JID for the Meta AI assistant.
var MetaJID = types.NewMetaAIJID

var metaMu sync.Mutex

// pendingReplies maps the Meta AI JID string to the chat JID that issued the query.
var pendingReplies = make(map[string]types.JID)

// lastProcessedResponse tracks the last text seen for a given Meta AI response ID
// so we only forward longer (more complete) streaming updates.
var lastProcessedResponse = make(map[string]string)

// sentMessageIDs maps a Meta AI response ID to the message ID we already sent,
// so we can edit it in place as the streamed response grows.
var sentMessageIDs = make(map[string]types.MessageID)

// HandleMetaAIResponse processes incoming messages from the Meta AI JID and
// forwards the response text (or edits the previous message) back to the
// original requester's chat.
func HandleMetaAIResponse(client *whatsmeow.Client, v *events.Message) {
	var responseText string
	resID := v.Message.GetMessageContextInfo().GetBotMetadata().GetBotResponseID()

	// Check for image response from Meta AI (e.g. image generation).
	if img := v.Message.GetImageMessage(); img != nil {
		metaMu.Lock()
		targetJID, ok := pendingReplies[v.Info.Sender.String()]
		metaMu.Unlock()
		if ok {
			client.SendMessage(context.Background(), targetJID, &waProto.Message{ImageMessage: img}) //nolint:errcheck
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

			// Case 1: the message itself is an image or video (caption = ".meta query").
			if img := ctx.Event.Message.GetImageMessage(); img != nil {
				img.Caption = proto.String(query)
				outMsg = &waProto.Message{ImageMessage: img}
			} else if vid := ctx.Event.Message.GetVideoMessage(); vid != nil {
				vid.Caption = proto.String(query)
				outMsg = &waProto.Message{VideoMessage: vid}
			} else if ext := ctx.Event.Message.GetExtendedTextMessage(); ext != nil {
				// Case 2: the user replied to a media message with ".meta query".
				// Preserve the full ExtendedTextMessage (including contextInfo /
				// quotedMessage) so Meta AI can see the referenced media.
				// Just replace the visible text with the parsed query.
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

			// Case 3: plain text query (no media involved).
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

			// Store which chat to relay the response back to.
			metaMu.Lock()
			pendingReplies[MetaJID.String()] = ctx.Event.Info.Chat
			metaMu.Unlock()
			return nil
		},
	})
}
