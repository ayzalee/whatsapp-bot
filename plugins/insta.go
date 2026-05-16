package plugins

import (
"context"
"encoding/json"
"fmt"
"io"
"net/http"
"net/url"
"strings"

"go.mau.fi/whatsmeow"
waProto "go.mau.fi/whatsmeow/proto/waE2E"
"google.golang.org/protobuf/proto"
)

func init() {
Register(&Command{
Pattern:  "ig",
Category: "download",
Func:     instaCmd,
})
}

type instaResponse struct {
Status bool     `json:"status"`
Result []string `json:"result"`
Error  string   `json:"error_message"`
}

func instaCmd(ctx *Context) error {
link := strings.TrimSpace(ctx.Text)
if link == "" {
ctx.Reply(T().IgUsage)
return nil
}



apiURL := "https://api-25ca.onrender.com/api/instagram?url=" + url.QueryEscape(link)
resp, err := http.Get(apiURL)
if err != nil {
ctx.Reply(fmt.Sprintf("❌ Failed to fetch: %v", err))
return nil
}
defer resp.Body.Close()

body, _ := io.ReadAll(resp.Body)

var result instaResponse
if err := json.Unmarshal(body, &result); err != nil {
ctx.Reply("❌ Failed to parse response.")
return nil
}

if result.Error != "" {
ctx.Reply("❌ " + result.Error)
return nil
}

if !result.Status || len(result.Result) == 0 {
ctx.Reply(T().MediaNotFound)
return nil
}
videoURL := result.Result[0]

// Download video bytes
vResp, err := http.Get(videoURL)
if err != nil {
ctx.Reply(fmt.Sprintf("❌ Failed to download video: %v", err))
return nil
}
defer vResp.Body.Close()
videoBytes, err := io.ReadAll(vResp.Body)
if err != nil {
ctx.Reply(fmt.Sprintf("❌ Failed to read video: %v", err))
return nil
}

// Upload to WhatsApp
uploaded, err := ctx.Client.Upload(context.Background(), videoBytes, whatsmeow.MediaVideo)
if err != nil {
ctx.Reply(fmt.Sprintf("❌ Failed to upload: %v", err))
return nil
}

msgID := ctx.Event.Info.ID
senderJID := ctx.Event.Info.Sender.String()
chatJID := ctx.Event.Info.Chat.String()
msg := &waProto.Message{
VideoMessage: &waProto.VideoMessage{
URL:           proto.String(uploaded.URL),
DirectPath:    proto.String(uploaded.DirectPath),
MediaKey:      uploaded.MediaKey,
FileEncSHA256: uploaded.FileEncSHA256,
FileSHA256:    uploaded.FileSHA256,
FileLength:    proto.Uint64(uint64(len(videoBytes))),
Mimetype:      proto.String("video/mp4"),
ContextInfo: &waProto.ContextInfo{
StanzaID:      proto.String(msgID),
Participant:   proto.String(senderJID),
QuotedMessage: &waProto.Message{Conversation: proto.String("")},
RemoteJID:     proto.String(chatJID),
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
return nil
}
