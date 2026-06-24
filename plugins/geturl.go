package plugins

import (
"bytes"
"context"
"io"
"mime/multipart"
"net/http"
"strings"
)

var mimeExtMap = map[string]string{
"image/jpeg":       "jpg",
"image/png":        "png",
"image/gif":        "gif",
"image/webp":       "webp",
"video/mp4":        "mp4",
"video/3gpp":       "3gp",
"audio/mpeg":       "mp3",
"audio/ogg":        "ogg",
"application/pdf":  "pdf",
"application/zip":  "zip",
}

func init() {
Register(&Command{
Pattern:  "url",
Category: "media",
Func: func(ctx *Context) error {
ci := ctx.Event.Message.GetExtendedTextMessage().GetContextInfo()
if ci == nil || ci.GetQuotedMessage() == nil {
ctx.Reply("Reply any media")
return nil
}
quoted := ci.GetQuotedMessage()

var data []byte
var err error
var mimetype string

switch {
case quoted.GetImageMessage() != nil:
img := quoted.GetImageMessage()
data, err = ctx.Client.Download(context.Background(), img)
mimetype = img.GetMimetype()
case quoted.GetVideoMessage() != nil:
vid := quoted.GetVideoMessage()
data, err = ctx.Client.Download(context.Background(), vid)
mimetype = vid.GetMimetype()
case quoted.GetAudioMessage() != nil:
aud := quoted.GetAudioMessage()
data, err = ctx.Client.Download(context.Background(), aud)
mimetype = aud.GetMimetype()
case quoted.GetDocumentMessage() != nil:
doc := quoted.GetDocumentMessage()
data, err = ctx.Client.Download(context.Background(), doc)
mimetype = doc.GetMimetype()
case quoted.GetStickerMessage() != nil:
stk := quoted.GetStickerMessage()
data, err = ctx.Client.Download(context.Background(), stk)
mimetype = stk.GetMimetype()
default:
ctx.Reply("Error")
return nil
}

if err != nil {
ctx.Reply("Error")
return nil
}

ext := mimeExtMap[strings.Split(mimetype, ";")[0]]
if ext == "" {
ext = "bin"
}
filename := "file." + ext

url, err := uploadToCatbox(data, filename)
if err != nil {
ctx.Reply("Error")
return nil
}

ctx.Reply(url)
return nil
},
})
}

func uploadToCatbox(data []byte, filename string) (string, error) {
body := &bytes.Buffer{}
writer := multipart.NewWriter(body)

writer.WriteField("reqtype", "fileupload")

part, err := writer.CreateFormFile("fileToUpload", filename)
if err != nil {
return "", err
}
if _, err := part.Write(data); err != nil {
return "", err
}
writer.Close()

req, err := http.NewRequest("POST", "https://catbox.moe/user/api.php", body)
if err != nil {
return "", err
}
req.Header.Set("Content-Type", writer.FormDataContentType())

resp, err := http.DefaultClient.Do(req)
if err != nil {
return "", err
}
defer resp.Body.Close()

respBody, err := io.ReadAll(resp.Body)
if err != nil {
return "", err
}

return strings.TrimSpace(string(respBody)), nil
}
