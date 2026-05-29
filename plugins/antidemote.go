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
}

func HandleGroupParticipantChange(client *whatsmeow.Client, evt *events.GroupInfo) {
chatJID := evt.JID.String()

if !antiModChats[chatJID] {
return
}

botUser := client.Store.ID.User
actor := evt.Sender

// Ignore bot's own actions
if actor.User == botUser {
return
}

// Ignore sudo users
if BotSettings.IsSudo(actor.User) {
return
}

actorJIDStr := actor.ToNonAD().String()
var mentions []string
mentions = append(mentions, actorJIDStr)

if len(evt.Demote) > 0 {
// Re-promote demoted admins
var promoted []string
for _, jid := range evt.Demote {
if jid.User == botUser {
continue
}
client.UpdateGroupParticipants(context.Background(), evt.JID,
[]types.JID{jid}, whatsmeow.ParticipantChangePromote)
promoted = append(promoted, jid.User)
mentions = append(mentions, jid.ToNonAD().String())
}

// Demote the actor
client.UpdateGroupParticipants(context.Background(), evt.JID,
[]types.JID{actor.ToNonAD()}, whatsmeow.ParticipantChangeDemote)

msg := fmt.Sprintf(T().AntiModDemote, actor.User)
sendMentionMsg(client, evt.JID, msg, mentions)
}

if len(evt.Promote) > 0 {
// De-promote promoted users
for _, jid := range evt.Promote {
if jid.User == botUser {
continue
}
client.UpdateGroupParticipants(context.Background(), evt.JID,
[]types.JID{jid}, whatsmeow.ParticipantChangeDemote)
mentions = append(mentions, jid.ToNonAD().String())
}

// Demote the actor
client.UpdateGroupParticipants(context.Background(), evt.JID,
[]types.JID{actor.ToNonAD()}, whatsmeow.ParticipantChangeDemote)

msg := fmt.Sprintf(T().AntiModPromote, actor.User)
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
