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

const sendTimeout = 20 * time.Second

type sendTask struct {
	client *whatsmeow.Client
	to     types.JID
	msg    *waProto.Message
	id     types.MessageID
}

var sendQueue = make(chan sendTask, 512)

func init() {
	go sendWorker()
}

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
