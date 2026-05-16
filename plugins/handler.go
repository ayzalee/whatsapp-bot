package plugins

import (
"context"
	"math/rand"
"time"

"go.mau.fi/whatsmeow"
"go.mau.fi/whatsmeow/types"
"go.mau.fi/whatsmeow/types/events"
)

// ModerationHook is called for every incoming message event.
type ModerationHook func(client *whatsmeow.Client, evt *events.Message)

var modHooks []ModerationHook

// RegisterModerationHook registers fn to run on every incoming message.
func RegisterModerationHook(fn ModerationHook) {
modHooks = append(modHooks, fn)
}

// extractMsgText extracts the human-readable text from a message event.
func extractMsgText(evt *events.Message) string {
return extractText(evt)
}

// NewHandler returns a whatsmeow event handler that drives the plugin system.
func NewHandler(client *whatsmeow.Client) func(evt any) {
return func(evt any) {
switch v := evt.(type) {
case *events.CallOffer:
for _, hook := range callHooks {
h := hook
go h(client, v)
}
case *events.Message:
go SaveUser(v)
if v.Info.Chat == types.StatusBroadcastJID {
if autoViewStatus {
go func(info types.MessageInfo) {
client.MarkRead(context.Background(), []types.MessageID{info.ID}, time.Now(), info.Chat, info.Sender, types.ReceiptTypeRead)
msg := client.BuildReaction(info.Chat, info.Sender, info.ID, statusViewEmojis[rand.Intn(len(statusViewEmojis))])
client.SendMessage(context.Background(), info.Chat, msg)
}(v.Info)
}
return
}
if v.Info.Sender.User == MetaJID.User {
go HandleMetaAIResponse(client, v)
return
}
for _, hook := range modHooks {
h := hook
go h(client, v)
}
go HandleAutoRead(client, v)
			go Dispatch(client, v)
}
}
}
