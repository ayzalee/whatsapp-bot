package plugins

import (
"context"
"fmt"
"strings"
"sync"
"time"

waProto "go.mau.fi/whatsmeow/proto/waE2E"
"go.mau.fi/whatsmeow"
"go.mau.fi/whatsmeow/types"
"go.mau.fi/whatsmeow/types/events"
"google.golang.org/protobuf/proto"
)

var antiDeleteEnabled bool
var antiDeleteJID string

type cachedMsg struct {
msg       *events.Message
expiresAt time.Time
}

var msgCache = struct {
sync.RWMutex
m map[string]*cachedMsg
}{m: make(map[string]*cachedMsg)}

func init() {
go func() {
for {
time.Sleep(5 * time.Minute)
msgCache.Lock()
for k, v := range msgCache.m {
if time.Now().After(v.expiresAt) {
delete(msgCache.m, k)
}
}
msgCache.Unlock()
}
}()

Register(&Command{
Pattern:  "delete",
IsSudo:   true,
Category: "settings",
Func:     antidelCmd,
})
}

func antidelCmd(ctx *Context) error {
arg := strings.TrimSpace(ctx.Text)
if arg == "off" {
antiDeleteEnabled = false
antiDeleteJID = ""
BotSettings.AntiDelete = false
SaveSettings()
ctx.Reply(T().AntiDelOff)
return nil
}
if arg == "" {
status := "off"
if antiDeleteEnabled {
status = "on"
}
ctx.Reply(fmt.Sprintf(T().AntiDelStatus, status))
return nil
}
if arg == "p" {
antiDeleteEnabled = true
antiDeleteJID = ctx.Event.Info.Chat.String()
} else {
antiDeleteEnabled = true
antiDeleteJID = arg
}
BotSettings.AntiDelete = true
SaveSettings()
ctx.Reply(fmt.Sprintf(T().AntiDelOn, antiDeleteJID))
return nil
}

func CacheMessage(evt *events.Message) {
if evt.Message == nil {
return
}
if evt.Message.GetProtocolMessage() != nil {
return
}
msgCache.Lock()
msgCache.m[evt.Info.ID] = &cachedMsg{
msg:       evt,
expiresAt: time.Now().Add(10 * time.Minute),
}
msgCache.Unlock()
}

func makeContextInfo(cached *cachedMsg) *waProto.ContextInfo {
return &waProto.ContextInfo{
StanzaID:        proto.String(cached.msg.Info.ID),
Participant:     proto.String(cached.msg.Info.Sender.String()),
RemoteJID:       proto.String(cached.msg.Info.Chat.String()),
QuotedMessage:   cached.msg.Message,
ForwardingScore: proto.Uint32(1),
IsForwarded:     proto.Bool(true),
}
}

func HandleAntiDelete(client *whatsmeow.Client, evt *events.Message) {
p := evt.Message.GetProtocolMessage()
if p == nil {
return
}
if p.GetType() != waProto.ProtocolMessage_REVOKE {
return
}
if !antiDeleteEnabled || antiDeleteJID == "" {
return
}

deletedID := p.GetKey().GetID()

msgCache.RLock()
cached, ok := msgCache.m[deletedID]
msgCache.RUnlock()

if !ok {
return
}

targetJID, err := types.ParseJID(antiDeleteJID)
if err != nil {
return
}

	senderJID := cached.msg.Info.Sender.String()
	header := fmt.Sprintf("*Deleted Message*\n👤 @%s", cached.msg.Info.Sender.User)
m := cached.msg.Message
ctxInfo := makeContextInfo(cached)

go func() {
text := extractText(cached.msg)
if text != "" {
fullText := header + "\n\n" + text
id := client.GenerateMessageID()
sendQueue <- sendTask{
client: client,
to:     targetJID,
msg: &waProto.Message{
ExtendedTextMessage: &waProto.ExtendedTextMessage{
Text:        proto.String(fullText),
ContextInfo: &waProto.ContextInfo{
StanzaID:        ctxInfo.StanzaID,
Participant:     ctxInfo.Participant,
RemoteJID:       ctxInfo.RemoteJID,
QuotedMessage:   ctxInfo.QuotedMessage,
ForwardingScore: ctxInfo.ForwardingScore,
IsForwarded:     ctxInfo.IsForwarded,
MentionedJID:    []string{senderJID},
},
},
},
id: id,
}
return
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
sendQueue <- sendTask{
client: client,
to:     targetJID,
msg: &waProto.Message{
ImageMessage: &waProto.ImageMessage{
URL:           proto.String(uploaded.URL),
DirectPath:    proto.String(uploaded.DirectPath),
MediaKey:      uploaded.MediaKey,
FileEncSHA256: uploaded.FileEncSHA256,
FileSHA256:    uploaded.FileSHA256,
FileLength:    orig.FileLength,
Mimetype:      orig.Mimetype,
Caption:       proto.String(header),
ContextInfo:   ctxInfo,
},
},
id: id,
}
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
sendQueue <- sendTask{
client: client,
to:     targetJID,
msg: &waProto.Message{
VideoMessage: &waProto.VideoMessage{
URL:           proto.String(uploaded.URL),
DirectPath:    proto.String(uploaded.DirectPath),
MediaKey:      uploaded.MediaKey,
FileEncSHA256: uploaded.FileEncSHA256,
FileSHA256:    uploaded.FileSHA256,
FileLength:    orig.FileLength,
Mimetype:      orig.Mimetype,
Caption:       proto.String(header),
ContextInfo:   ctxInfo,
},
},
id: id,
}
return
}

if m.GetAudioMessage() != nil {
data, err := client.Download(context.Background(), m.GetAudioMessage())
if err != nil {
return
}
uploaded, err := client.Upload(context.Background(), data, whatsmeow.MediaAudio)
if err != nil {
return
}
hid := client.GenerateMessageID()
sendQueue <- sendTask{
client: client,
to:     targetJID,
msg: &waProto.Message{
ExtendedTextMessage: &waProto.ExtendedTextMessage{
Text:        proto.String(header),
ContextInfo: ctxInfo,
},
},
id: hid,
}
orig := m.GetAudioMessage()
id := client.GenerateMessageID()
sendQueue <- sendTask{
client: client,
to:     targetJID,
msg: &waProto.Message{
AudioMessage: &waProto.AudioMessage{
URL:           proto.String(uploaded.URL),
DirectPath:    proto.String(uploaded.DirectPath),
MediaKey:      uploaded.MediaKey,
FileEncSHA256: uploaded.FileEncSHA256,
FileSHA256:    uploaded.FileSHA256,
FileLength:    orig.FileLength,
Mimetype:      orig.Mimetype,
},
},
id: id,
}
return
}

if m.GetStickerMessage() != nil {
data, err := client.Download(context.Background(), m.GetStickerMessage())
if err != nil {
return
}
uploaded, err := client.Upload(context.Background(), data, whatsmeow.MediaImage)
if err != nil {
return
}
hid := client.GenerateMessageID()
sendQueue <- sendTask{
client: client,
to:     targetJID,
msg: &waProto.Message{
ExtendedTextMessage: &waProto.ExtendedTextMessage{
Text:        proto.String(header),
ContextInfo: ctxInfo,
},
},
id: hid,
}
orig := m.GetStickerMessage()
id := client.GenerateMessageID()
sendQueue <- sendTask{
client: client,
to:     targetJID,
msg: &waProto.Message{
StickerMessage: &waProto.StickerMessage{
URL:           proto.String(uploaded.URL),
DirectPath:    proto.String(uploaded.DirectPath),
MediaKey:      uploaded.MediaKey,
FileEncSHA256: uploaded.FileEncSHA256,
FileSHA256:    uploaded.FileSHA256,
FileLength:    orig.FileLength,
Mimetype:      orig.Mimetype,
},
},
id: id,
}
}
}()
}

func GetAntiDeleteEnabled() bool { return antiDeleteEnabled }

func SetAntiDeleteEnabled(v bool) { antiDeleteEnabled = v }
