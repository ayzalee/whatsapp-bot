package plugins

import (
"bytes"
"encoding/json"
"fmt"
"io"
"mime/multipart"
"net/http"
"os"
"strings"
)

func init() {
Register(&Command{
Pattern:  "antiporn",
IsGroup:  true,
IsAdmin:  true,
Category: "group",
Func: func(ctx *Context) error {
chatJID := ctx.Event.Info.Chat.String()
arg := strings.ToLower(strings.TrimSpace(ctx.Text))
switch arg {
case "on":
if os.Getenv("SIGHTENGINE_USER") == "" || os.Getenv("SIGHTENGINE_SECRET") == "" {
ctx.Reply(T().AntipornNoCreds)
return nil
}
setAntipornEnabled(chatJID, true)
ctx.Reply(T().AntipornOn)
case "off":
setAntipornEnabled(chatJID, false)
ctx.Reply(T().AntipornOff)
default:
status := "off"
if getAntipornEnabled(chatJID) {
status = "on"
}
ctx.Reply(fmt.Sprintf(T().AntipornStatus, status))
}
return nil
},
})
}

type nudityResult struct {
Nudity struct {
SexualActivity float64 `json:"sexual_activity"`
SexualDisplay  float64 `json:"sexual_display"`
Erotica        float64 `json:"erotica"`
} `json:"nudity"`
Status string `json:"status"`
}

func checkNudity(imageBytes []byte) (float64, error) {
apiUser := os.Getenv("SIGHTENGINE_USER")
apiSecret := os.Getenv("SIGHTENGINE_SECRET")
if apiUser == "" || apiSecret == "" {
return 0, fmt.Errorf("sightengine credentials not configured")
}

body := &bytes.Buffer{}
writer := multipart.NewWriter(body)

part, err := writer.CreateFormFile("media", "image.jpg")
if err != nil {
return 0, err
}
if _, err := part.Write(imageBytes); err != nil {
return 0, err
}
writer.WriteField("models", "nudity-2.1")
writer.WriteField("api_user", apiUser)
writer.WriteField("api_secret", apiSecret)
writer.Close()

req, err := http.NewRequest("POST", "https://api.sightengine.com/1.0/check.json", body)
if err != nil {
return 0, err
}
req.Header.Set("Content-Type", writer.FormDataContentType())

resp, err := http.DefaultClient.Do(req)
if err != nil {
return 0, err
}
defer resp.Body.Close()

data, err := io.ReadAll(resp.Body)
if err != nil {
return 0, err
}

var result nudityResult
if err := json.Unmarshal(data, &result); err != nil {
return 0, err
}
if result.Status != "success" {
return 0, fmt.Errorf("sightengine error: %s", string(data))
}

score := result.Nudity.SexualActivity + result.Nudity.SexualDisplay + result.Nudity.Erotica
return score, nil
}
