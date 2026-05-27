package plugins

import (
"fmt"
	"time"
"os"
"strings"
)

func init() {
Register(&Command{
Pattern:  "setvar",
IsSudo:   true,
Category: "owner",
Func: func(ctx *Context) error {
arg := strings.TrimSpace(ctx.Text)
if arg == "" {
ctx.Reply(T().SetVarUsage)
return nil
}
parts := strings.SplitN(arg, "=", 2)
if len(parts) != 2 {
ctx.Reply(T().SetVarInvalid)
return nil
}
key := strings.ToUpper(strings.TrimSpace(parts[0]))
value := strings.TrimSpace(parts[1])
if err := updateEnvFile(key, value); err != nil {
ctx.Reply("Failed to update: " + err.Error())
return nil
}
os.Setenv(key, value)
applySettingFromEnv(key, value)
ctx.Reply(fmt.Sprintf(T().SetVarOK, key, value))
return nil
},
})

Register(&Command{
Pattern:  "getvar",
IsSudo:   true,
Category: "owner",
Func: func(ctx *Context) error {
data, err := os.ReadFile(".env")
if err != nil {
ctx.Reply(T().GetVarFailed)
return nil
}
var sb strings.Builder
sb.WriteString("*Environment Variables:*\n\n")
for _, line := range strings.Split(string(data), "\n") {
line = strings.TrimSpace(line)
if line == "" || strings.HasPrefix(line, "#") {
continue
}
sb.WriteString("`" + line + "`\n")
}
ctx.Reply(strings.TrimRight(sb.String(), "\n"))
return nil
},
})

Register(&Command{
Pattern:  "delvar",
IsSudo:   true,
Category: "owner",
Func: func(ctx *Context) error {
key := strings.ToUpper(strings.TrimSpace(ctx.Text))
if key == "" {
ctx.Reply(T().DelVarUsage)
return nil
}
if err := updateEnvFile(key, ""); err != nil {
ctx.Reply("Failed: " + err.Error())
return nil
}
os.Unsetenv(key)
ctx.Reply(fmt.Sprintf(T().DelVarOK, key))
return nil
},
})
}

func updateEnvFile(key, value string) error {
data, err := os.ReadFile(".env")
if err != nil {
data = []byte{}
}
lines := strings.Split(string(data), "\n")
found := false
for i, line := range lines {
trimmed := strings.TrimSpace(line)
if strings.HasPrefix(trimmed, "#") || trimmed == "" {
continue
}
parts := strings.SplitN(trimmed, "=", 2)
if len(parts) < 1 {
continue
}
if strings.TrimSpace(parts[0]) == key {
if value == "" {
lines = append(lines[:i], lines[i+1:]...)
} else {
lines[i] = key + "=" + value
}
found = true
break
}
}
if !found && value != "" {
lines = append(lines, key+"="+value)
}
result := strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
return os.WriteFile(".env", []byte(result), 0644)
}

func applySettingFromEnv(key, value string) {
	if key == "TZ" {
		if loc, err := time.LoadLocation(value); err == nil {
			time.Local = loc
		}
		return
	}
on := value == "true"
switch key {
case "ALWAYS_ONLINE":
BotSettings.AlwaysOnline = on
if on {
StartOnlineLoop(nil)
} else {
StopOnlineLoop()
}
SaveSettings()
case "AUTO_STATUS_VIEW":
BotSettings.AutoStatusView = on
SetAutoViewStatus(on)
SaveSettings()
case "CALL_REJECT":
BotSettings.CallReject = on
SetCallReject(on)
SaveSettings()
case "ANTI_DELETE":
BotSettings.AntiDelete = on
SetAntiDeleteEnabled(on)
SaveSettings()
case "READ_MSGS":
BotSettings.AutoRead = on
SetAutoReadEnabled(on)
SaveSettings()
case "BOT_MODE":
BotSettings.SetMode(Mode(value))
SaveSettings()
case "BOT_LANG":
BotSettings.SetLanguage(value)
case "BOT_PREFIX":
BotSettings.SetPrefixes(value)
SaveSettings()
}
}
