package plugins

import (
"context"
"fmt"
"strings"

"go.mau.fi/whatsmeow"
"go.mau.fi/whatsmeow/types"
"go.mau.fi/whatsmeow/types/events"
)

var antiGroupCallChats = map[string]bool{}

func init() {
Register(&Command{
Pattern:  "antigroupcall",
Aliases:  []string{"agc"},
IsGroup:  true,
IsAdmin:  true,
Category: "group",
Func: func(ctx *Context) error {
chatJID := ctx.Event.Info.Chat.String()
arg := strings.ToLower(strings.TrimSpace(ctx.Text))
switch arg {
case "on":
antiGroupCallChats[chatJID] = true
ctx.Reply("Anti group call enabled.")
case "off":
delete(antiGroupCallChats, chatJID)
ctx.Reply("Anti group call disabled.")
default:
status := "off"
if antiGroupCallChats[chatJID] {
status = "on"
}
ctx.Reply("*Anti Group Call*\nStatus: " + status + "\n\n.antigroupcall on\n.antigroupcall off")
}
return nil
},
})
}

func HandleGroupCallNotice(client *whatsmeow.Client, evt *events.CallOfferNotice) {
}

func HandleGroupCallMessage(client *whatsmeow.Client, evt *events.Message) {
if !evt.Info.IsGroup {
return
}

callLog := evt.Message.GetCallLogMesssage()
if callLog == nil {
return
}

fmt.Printf("[CALL] group=%s callType=%d callOutcome=%d sender=%s\n",
evt.Info.Chat.String(),
callLog.GetCallType().Number(),
callLog.GetCallOutcome().Number(),
evt.Info.Sender.User,
)

chatJID := evt.Info.Chat.String()
if !antiGroupCallChats[chatJID] {
return
}

sender := evt.Info.Sender.ToNonAD()
if BotSettings.IsSudo(sender.User) {
return
}

client.UpdateGroupParticipants(context.Background(), evt.Info.Chat,
[]types.JID{sender}, whatsmeow.ParticipantChangeRemove)
}
