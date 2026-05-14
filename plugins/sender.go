package plugins

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// sendTimeout caps the server-ACK wait in the queue worker.
// 20 s is well above normal WhatsApp RTT but prevents a single stuck send
// from holding messageSendLock for the default 75 s.
const sendTimeout = 20 * time.Second

type sendTask struct {
	client *whatsmeow.Client
	to     types.JID
	msg    *waProto.Message
	id     types.MessageID
}

// sendQueue buffers fire-and-forget outgoing messages.
// Capacity 512 ensures a burst of concurrent commands never blocks a handler.
var sendQueue = make(chan sendTask, 512)

func init() {
	go sendWorker()
}

// sendWorker drains sendQueue sequentially.
// Serialising all sends through one goroutine respects the Signal-session
// ordering constraint (whatsmeow's messageSendLock) without blocking callers.
func sendWorker() {
	for task := range sendQueue {
		_, err := task.client.SendMessage(
			context.Background(),
			task.to,
			task.msg,
			whatsmeow.SendRequestExtra{
				ID:      task.id,
				Timeout: sendTimeout,
			},
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[Send ERROR] %s → %s: %v\n", task.id, task.to, err)
		}
	}
}

// sendMention sends a text message with @mentions.
func sendMention(ctx *Context, text string, jids []string) {
	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: jids,
			},
		},
	}
	id := ctx.Client.GenerateMessageID()
	sendQueue <- sendTask{
		client: ctx.Client,
		to:     ctx.Event.Info.Chat,
		msg:    msg,
		id:     id,
	}
}
