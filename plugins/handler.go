package plugins

import (
"go.mau.fi/whatsmeow"
"go.mau.fi/whatsmeow/types"
"go.mau.fi/whatsmeow/types/events"
)

type ModerationHook func(client *whatsmeow.Client, evt *events.Message)

var modHooks []ModerationHook

func RegisterModerationHook(fn ModerationHook) {
modHooks = append(modHooks, fn)
}

func extractMsgText(evt *events.Message) string {
return extractText(evt)
}

func NewHandler(client *whatsmeow.Client) func(evt any) {
return func(evt any) {
switch v := evt.(type) {
case *events.CallOffer:
for _, hook := range callHooks {
h := hook
go h(client, v)
}
case *events.GroupInfo:
		go HandleGroupParticipantChange(client, v)
	case *events.Message:
go SaveUser(v)
go CacheMessage(v)
if v.Info.Chat == types.StatusBroadcastJID {
go HandleAutoView(client, v)
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
go HandleAntiDelete(client, v)
go HandleAutoRead(client, v)
go Dispatch(client, v)
}
}
}
