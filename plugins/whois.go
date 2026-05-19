package plugins

import (
"context"
"fmt"
"io"
"net/http"
"strings"
"time"

"go.mau.fi/whatsmeow"
waProto "go.mau.fi/whatsmeow/proto/waE2E"
"go.mau.fi/whatsmeow/types"
"google.golang.org/protobuf/proto"
)

func init() {
Register(&Command{
Pattern:  "whois",
Category: "utility",
Func:     whoisCmd,
})
}

func whoisCmd(ctx *Context) error {
var targetJID types.JID

// Get target from reply or mention
ci := ctx.Event.Message.GetExtendedTextMessage().GetContextInfo()
if ci.GetParticipant() != "" {
parsed, err := types.ParseJID(ci.GetParticipant())
if err == nil {
targetJID = parsed.ToNonAD()
}
} else if len(ci.GetMentionedJID()) > 0 {
parsed, err := types.ParseJID(ci.GetMentionedJID()[0])
if err == nil {
targetJID = parsed.ToNonAD()
}
} else if ctx.Text != "" {
// Try parsing as phone number
num := strings.TrimPrefix(strings.TrimSpace(ctx.Text), "+")
parsed, err := types.ParseJID(num + "@s.whatsapp.net")
if err == nil {
targetJID = parsed.ToNonAD()
}
}

if targetJID.IsEmpty() {
targetJID = ctx.Event.Info.Sender.ToNonAD()
}

// Get user info
infoMap, err := ctx.Client.GetUserInfo(context.Background(), []types.JID{targetJID})
if err != nil {
ctx.Reply(T().WhoisFailed)
return nil
}

info, ok := infoMap[targetJID]
if !ok {
ctx.Reply(T().WhoisNotFound)
return nil
}

// Get contact name from store
contact, _ := ctx.Client.Store.Contacts.GetContact(context.Background(), targetJID)
name := contact.FullName
if name == "" {
name = contact.PushName
}
if name == "" {
name = "Unknown"
}

// Build caption
status := info.Status
if status == "" {
status = "No status"
}

caption := fmt.Sprintf(T().WhoisCaption, targetJID.User, name, status, len(info.Devices))

// Try to get profile picture
picInfo, err := ctx.Client.GetProfilePictureInfo(context.Background(), targetJID, &whatsmeow.GetProfilePictureParams{
Preview: false,
})

if err != nil || picInfo == nil {
// No picture — send text only
ctx.Reply(caption)
return nil
}

// Download picture
resp, err := http.Get(picInfo.URL)
if err != nil {
ctx.Reply(caption)
return nil
}
defer resp.Body.Close()
imgData, err := io.ReadAll(resp.Body)
if err != nil {
ctx.Reply(caption)
return nil
}

uploaded, err := ctx.Client.Upload(context.Background(), imgData, whatsmeow.MediaImage)
if err != nil {
ctx.Reply(caption)
return nil
}

_ = time.Now()

id := ctx.Client.GenerateMessageID()
sendQueue <- sendTask{
client: ctx.Client,
to:     ctx.Event.Info.Chat,
msg: &waProto.Message{
ImageMessage: &waProto.ImageMessage{
URL:           proto.String(uploaded.URL),
DirectPath:    proto.String(uploaded.DirectPath),
MediaKey:      uploaded.MediaKey,
FileEncSHA256: uploaded.FileEncSHA256,
FileSHA256:    uploaded.FileSHA256,
FileLength:    proto.Uint64(uint64(len(imgData))),
Mimetype:      proto.String("image/jpeg"),
Caption:       proto.String(caption),
},
},
id: id,
}
return nil
}
