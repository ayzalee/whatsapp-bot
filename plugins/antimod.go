package plugins

import (
"context"
"fmt"
"strings"

"go.mau.fi/whatsmeow"
waProto "go.mau.fi/whatsmeow/proto/waE2E"
"go.mau.fi/whatsmeow/types"
"go.mau.fi/whatsmeow/types/events"
"google.golang.org/protobuf/proto"
)

var antiModChats = map[string]bool{}
var pdmChats = map[string]bool{}

func init() {
Register(&Command{
Pattern:  "antimod",
IsGroup:  true,
IsAdmin:  true,
Category: "group",
Func: func(ctx *Context) error {
chatJID := ctx.Event.Info.Chat.String()
arg := strings.ToLower(strings.TrimSpace(ctx.Text))
switch arg {
case "on":
antiModChats[chatJID] = true
ctx.Reply(T().AntiModOn)
case "off":
delete(antiModChats, chatJID)
ctx.Reply(T().AntiModOff)
default:
status := "off"
if antiModChats[chatJID] {
status = "on"
}
ctx.Reply(fmt.Sprintf(T().AntiModStatus, status))
}
return nil
},
})

Register(&Command{
Pattern:  "pdm",
IsGroup:  true,
IsAdmin:  true,
Category: "group",
Func: func(ctx *Context) error {
chatJID := ctx.Event.Info.Chat.String()
arg := strings.ToLower(strings.TrimSpace(ctx.Text))
switch arg {
case "on":
pdmChats[chatJID] = true
ctx.Reply("Promote/demote notifications enabled.")
case "off":
delete(pdmChats, chatJID)
ctx.Reply("Promote/demote notifications disabled.")
default:
status := "off"
if pdmChats[chatJID] {
status = "on"
}
ctx.Reply("*PDM Notifications*\nStatus: " + status + "\n\n.pdm on\n.pdm off")
}
return nil
},
})
}

func HandleGroupParticipantChange(client *whatsmeow.Client, evt *events.GroupInfo) {
chatJID := evt.JID.String()
botUser := client.Store.ID.User
actor := evt.Sender

if actor.User == botUser {
return
}

actorJIDStr := actor.ToNonAD().String()
actorStr := actor.User

// PDM notifications
if pdmChats[chatJID] {
for _, jid := range evt.Promote {
msg := fmt.Sprintf("_@%s promoted @%s_", actorStr, jid.User)
sendMentionMsg(client, evt.JID, msg, []string{actorJIDStr, jid.ToNonAD().String()})
}
for _, jid := range evt.Demote {
msg := fmt.Sprintf("_@%s demoted @%s_", actorStr, jid.User)
sendMentionMsg(client, evt.JID, msg, []string{actorJIDStr, jid.ToNonAD().String()})
}
}

// Antimod protection
if !antiModChats[chatJID] {
return
}

if BotSettings.IsSudo(actor.User) {
return
}

var mentions []string
mentions = append(mentions, actorJIDStr)

if len(evt.Demote) > 0 {
for _, jid := range evt.Demote {
if jid.User == botUser {
continue
}
client.UpdateGroupParticipants(context.Background(), evt.JID,
[]types.JID{jid}, whatsmeow.ParticipantChangePromote)
mentions = append(mentions, jid.ToNonAD().String())
}
client.UpdateGroupParticipants(context.Background(), evt.JID,
[]types.JID{actor.ToNonAD()}, whatsmeow.ParticipantChangeDemote)
msg := fmt.Sprintf(T().AntiModDemote, actorStr)
sendMentionMsg(client, evt.JID, msg, mentions)
}

if len(evt.Promote) > 0 {
for _, jid := range evt.Promote {
if jid.User == botUser {
continue
}
client.UpdateGroupParticipants(context.Background(), evt.JID,
[]types.JID{jid}, whatsmeow.ParticipantChangeDemote)
mentions = append(mentions, jid.ToNonAD().String())
}
client.UpdateGroupParticipants(context.Background(), evt.JID,
[]types.JID{actor.ToNonAD()}, whatsmeow.ParticipantChangeDemote)
msg := fmt.Sprintf(T().AntiModPromote, actorStr)
sendMentionMsg(client, evt.JID, msg, mentions)
}
}

func sendMentionMsg(client *whatsmeow.Client, chat types.JID, text string, mentions []string) {
id := client.GenerateMessageID()
sendQueue <- sendTask{
client: client,
to:     chat,
msg: &waProto.Message{
ExtendedTextMessage: &waProto.ExtendedTextMessage{
Text: proto.String(text),
ContextInfo: &waProto.ContextInfo{
MentionedJID: mentions,
},
},
},
id: id,
}
}
