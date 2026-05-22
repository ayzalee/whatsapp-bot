package plugins

import (
"encoding/json"
"fmt"
"io"
"net/http"
"os"
"os/exec"
"path/filepath"
"strings"
)

func init() {
Register(&Command{
Pattern:  "plugin",
Aliases:  []string{"remove"},
IsSudo:   true,
Category: "owner",
Func:     pluginCmd,
})
Register(&Command{
Pattern:  "plugins",
IsSudo:   true,
Category: "owner",
Func:     listPluginsCmd,
})
}

type gistFile struct {
Filename string `json:"filename"`
Content  string `json:"content"`
}

type gistResponse struct {
Files map[string]gistFile `json:"files"`
}

func fetchGist(gistID string) (filename, content string, err error) {
url := "https://api.github.com/gists/" + gistID
req, _ := http.NewRequest("GET", url, nil)
req.Header.Set("Accept", "application/vnd.github+json")
req.Header.Set("User-Agent", "zaelix-bot")
resp, err := http.DefaultClient.Do(req)
if err != nil {
return "", "", err
}
defer resp.Body.Close()
body, err := io.ReadAll(resp.Body)
if err != nil {
return "", "", err
}
var gist gistResponse
if err = json.Unmarshal(body, &gist); err != nil {
return "", "", fmt.Errorf("invalid gist response: %w", err)
}
for name, file := range gist.Files {
if strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".txt") {
if strings.HasPrefix(strings.TrimSpace(file.Content), "package plugins") {
return file.Filename, file.Content, nil
}
}
}
return "", "", fmt.Errorf("no valid plugin file found in gist")
}

func extractGistID(input string) string {
input = strings.TrimSpace(input)
input = strings.TrimPrefix(input, "https://")
input = strings.TrimPrefix(input, "http://")
parts := strings.Split(strings.TrimSuffix(input, "/"), "/")
return parts[len(parts)-1]
}

func pluginCmd(ctx *Context) error {
arg := strings.TrimSpace(ctx.Text)

// .remove <name>
if ctx.Matched == "remove" {
if arg == "" {
ctx.Reply(T().PluginRemoveUsage)
return nil
}
name := arg
if !strings.HasPrefix(name, "ext_") {
name = "ext_" + name
}
if !strings.HasSuffix(name, ".go") {
name += ".go"
}
pluginPath := filepath.Join("plugins", name)
if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
ctx.Reply(fmt.Sprintf(T().PluginNotFound, arg))
return nil
}
if err := os.Remove(pluginPath); err != nil {
ctx.Reply(fmt.Sprintf(T().PluginRemoveFail, err.Error()))
return nil
}
ctx.Reply(fmt.Sprintf(T().PluginRemoved, arg))
rebuildAndRestart(ctx)
return nil
}

// .plugin with no args
if arg == "" {
ctx.Reply(T().PluginManager)
return nil
}

ctx.Reply(T().PluginFetching)

var filename, content string
var err error

if strings.Contains(arg, "gist.github.com") || strings.Contains(arg, "gist.com") {
gistID := extractGistID(arg)
filename, content, err = fetchGist(gistID)
if err != nil {
ctx.Reply(fmt.Sprintf(T().PluginFetchFail, err.Error()))
return nil
}
} else {
rawURL := arg
if !strings.HasPrefix(rawURL, "http") {
rawURL = "https://" + rawURL
}
if strings.Contains(rawURL, "github.com") && strings.Contains(rawURL, "/blob/") {
rawURL = strings.Replace(rawURL, "github.com", "raw.githubusercontent.com", 1)
rawURL = strings.Replace(rawURL, "/blob/", "/", 1)
}
resp, err2 := http.Get(rawURL)
if err2 != nil {
ctx.Reply(fmt.Sprintf(T().PluginURLFail, err2.Error()))
return nil
}
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)
content = string(body)
parts := strings.Split(rawURL, "/")
filename = parts[len(parts)-1]
if !strings.HasSuffix(filename, ".go") {
filename += ".go"
}
}

if !strings.HasPrefix(strings.TrimSpace(content), "package plugins") {
ctx.Reply(T().PluginRejected)
return nil
}

pluginName := "ext_" + strings.TrimPrefix(strings.TrimSuffix(filepath.Base(filename), ".go"), "ext_")
pluginName = strings.TrimSuffix(pluginName, ".txt") + ".go"
pluginPath := filepath.Join("plugins", pluginName)

if err = os.WriteFile(pluginPath, []byte(content), 0600); err != nil {
ctx.Reply(fmt.Sprintf(T().PluginSaveFail, err.Error()))
return nil
}

ctx.Reply(fmt.Sprintf(T().PluginSaved, pluginName))
rebuildAndRestart(ctx)
return nil
}

func rebuildAndRestart(ctx *Context) {
exePath, _ := os.Executable()
tmpPath := exePath + ".new"
cmd := exec.Command("go", "build", "-o", tmpPath, ".")
cmd.Dir, _ = os.Getwd()
out, buildErr := cmd.CombinedOutput()
if buildErr != nil {
ctx.Reply(fmt.Sprintf(T().PluginBuildFail, string(out)))
return
}
if err := os.Rename(tmpPath, exePath); err != nil {
ctx.Reply(fmt.Sprintf(T().PluginBinaryFail, err.Error()))
return
}
ctx.Reply(T().PluginDone)
if restartFunc != nil {
go restartFunc()
}
}

func listPluginsCmd(ctx *Context) error {
entries, err := os.ReadDir("plugins")
if err != nil {
ctx.Reply(T().PluginDirFail)
return nil
}
var found []string
for _, e := range entries {
if strings.HasPrefix(e.Name(), "ext_") && strings.HasSuffix(e.Name(), ".go") {
name := strings.TrimPrefix(e.Name(), "ext_")
name = strings.TrimSuffix(name, ".go")
found = append(found, "  - "+name)
}
}
if len(found) == 0 {
ctx.Reply(T().PluginNone)
return nil
}
ctx.Reply(fmt.Sprintf(T().PluginList, strings.Join(found, "\n")))
return nil
}
