package plugins

import (
"context"
"fmt"
"strings"
"time"

"go.mau.fi/whatsmeow"
waProto "go.mau.fi/whatsmeow/proto/waE2E"
"go.mau.fi/whatsmeow/types"
"go.mau.fi/whatsmeow/types/events"
"google.golang.org/protobuf/proto"
)

var (
autoViewStatus   bool
statusForwardJID string
statusNoDL       bool              // don't download/forward media
statusExceptView []string          // skip viewing these JIDs
statusOnlyView   []string          // only view these JIDs
)

var statusViewEmojis = []string{"❤️", "🪻", "🤍", "🎐", "🌸", "💫"}

func statusHelp() string {
status := "off"
if autoViewStatus {
status = "on"
if statusForwardJID != "" {
status = "on + forwarding to " + statusForwardJID
}
if statusNoDL {
status += " (no-dl)"
}
}
return fmt.Sprintf(T().StatusInfo, status)
}

func init() {
Register(&Command{
Pattern:  "status",
IsSudo:   true,
Category: "settings",
Func: func(ctx *Context) error {
arg := strings.TrimSpace(ctx.Text)
lower := strings.ToLower(arg)

switch {
case lower == "on":
autoViewStatus = true
statusForwardJID = ""
BotSettings.AutoStatusView = true
SaveSettings()
ctx.Reply(T().StatusEnabled)

case lower == "off":
autoViewStatus = false
statusForwardJID = ""
statusNoDL = false
statusExceptView = nil
statusOnlyView = nil
BotSettings.AutoStatusView = false
SaveSettings()
ctx.Reply(T().StatusDisabled)

case lower == "no-dl":
autoViewStatus = true
statusNoDL = true
ctx.Reply(T().StatusNoDL)

case lower == "reset":
statusExceptView = nil
statusOnlyView = nil
statusNoDL = false
ctx.Reply(T().StatusReset)

case strings.HasPrefix(lower, "except-view "):
jids := strings.Split(strings.TrimPrefix(arg, "except-view "), ",")
statusExceptView = nil
for _, j := range jids {
j = strings.TrimSpace(j)
if j != "" {
if !strings.Contains(j, "@") {
j = j + "@s.whatsapp.net"
}
statusExceptView = append(statusExceptView, j)
}
}
autoViewStatus = true
ctx.Reply(fmt.Sprintf(T().StatusSkip, len(statusExceptView)))

case strings.HasPrefix(lower, "only-view "):
jids := strings.Split(strings.TrimPrefix(arg, "only-view "), ",")
statusOnlyView = nil
for _, j := range jids {
j = strings.TrimSpace(j)
if j != "" {
if !strings.Contains(j, "@") {
j = j + "@s.whatsapp.net"
}
statusOnlyView = append(statusOnlyView, j)
}
}
autoViewStatus = true
ctx.Reply(fmt.Sprintf(T().StatusOnly, len(statusOnlyView)))

case len(arg) > 5:
// Phone number or JID — enable + forward
jid := arg
if !strings.Contains(jid, "@") {
// If it looks like a group JID (long number) use @g.us, else @s.whatsapp.net
if len(jid) > 15 {
jid = jid + "@g.us"
} else {
jid = jid + "@s.whatsapp.net"
}
}
autoViewStatus = true
statusForwardJID = jid
ctx.Reply(fmt.Sprintf(T().StatusFwdTo, jid))

default:
ctx.Reply(statusHelp())
}
return nil
},
})
}

func HandleAutoView(client *whatsmeow.Client, evt *events.Message) {
if !autoViewStatus {
return
}

senderJID := evt.Info.Sender.String()

// Check only-view filter
if len(statusOnlyView) > 0 {
found := false
for _, j := range statusOnlyView {
if j == senderJID || strings.HasPrefix(senderJID, strings.TrimSuffix(j, "@s.whatsapp.net")) {
found = true
break
}
}
if !found {
return
}
}

// Check except-view filter
for _, j := range statusExceptView {
if j == senderJID || strings.HasPrefix(senderJID, strings.TrimSuffix(j, "@s.whatsapp.net")) {
return
}
}

// Mark as viewed
client.MarkRead(context.Background(), []types.MessageID{evt.Info.ID}, time.Now(), evt.Info.Chat, evt.Info.Sender, types.ReceiptTypeRead)

// React with random emoji
if len(statusViewEmojis) > 0 {
emojiIdx := time.Now().UnixNano() % int64(len(statusViewEmojis))
emoji := statusViewEmojis[emojiIdx]
reactMsg := client.BuildReaction(evt.Info.Chat, evt.Info.Sender, evt.Info.ID, emoji)
client.SendMessage(context.Background(), evt.Info.Chat, reactMsg)
}

// Forward if set and not no-dl
if statusForwardJID == "" || statusNoDL {
return
}

targetJID, err := types.ParseJID(statusForwardJID)
if err != nil {
return
}

m := evt.Message
senderName := evt.Info.PushName
if senderName == "" {
senderName = evt.Info.Sender.User
}

go func() {
contextInfo := &waProto.ContextInfo{
StanzaID:    proto.String(evt.Info.ID),
Participant: proto.String(evt.Info.Sender.String()),
RemoteJID:   proto.String(evt.Info.Chat.String()),
QuotedMessage: &waProto.Message{
Conversation: proto.String(senderName + " • Status"),
},
ForwardingScore: proto.Uint32(1),
IsForwarded:     proto.Bool(true),
}

if m.GetImageMessage() != nil {
data, err := client.Download(context.Background(), m.GetImageMessage())
if err != nil {
return
}
uploaded, err := client.Upload(context.Background(), data, whatsmeow.MediaImage)
if err != nil {
return
}
orig := m.GetImageMessage()
id := client.GenerateMessageID()
sendQueue <- sendTask{client: client, to: targetJID, msg: &waProto.Message{ImageMessage: &waProto.ImageMessage{
URL: proto.String(uploaded.URL), DirectPath: proto.String(uploaded.DirectPath),
MediaKey: uploaded.MediaKey, FileEncSHA256: uploaded.FileEncSHA256,
FileSHA256: uploaded.FileSHA256, FileLength: orig.FileLength,
Mimetype: orig.Mimetype, Caption: orig.Caption,
ContextInfo: contextInfo,
}}, id: id}
return
}

if m.GetVideoMessage() != nil {
data, err := client.Download(context.Background(), m.GetVideoMessage())
if err != nil {
return
}
uploaded, err := client.Upload(context.Background(), data, whatsmeow.MediaVideo)
if err != nil {
return
}
orig := m.GetVideoMessage()
id := client.GenerateMessageID()
sendQueue <- sendTask{client: client, to: targetJID, msg: &waProto.Message{VideoMessage: &waProto.VideoMessage{
URL: proto.String(uploaded.URL), DirectPath: proto.String(uploaded.DirectPath),
MediaKey: uploaded.MediaKey, FileEncSHA256: uploaded.FileEncSHA256,
FileSHA256: uploaded.FileSHA256, FileLength: orig.FileLength,
Mimetype: orig.Mimetype, Caption: orig.Caption,
ContextInfo: contextInfo,
}}, id: id}
return
}

text := extractText(evt)
if text != "" {
id := client.GenerateMessageID()
sendQueue <- sendTask{client: client, to: targetJID, msg: &waProto.Message{ExtendedTextMessage: &waProto.ExtendedTextMessage{
Text:        proto.String(text),
ContextInfo: contextInfo,
}}, id: id}
}
}()
}
