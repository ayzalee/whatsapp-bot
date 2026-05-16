package plugins

import (
"fmt"
"runtime"
"strings"
"time"
)

var botStartTime = time.Now()

var fancyMap = map[rune]string{
'0': "𝟶", '1': "𝟷", '2': "𝟸", '3': "𝟹", '4': "𝟺",
'5': "𝟻", '6': "𝟼", '7': "𝟽", '8': "𝟾", '9': "𝟿",
'a': "ᴀ", 'b': "ʙ", 'c': "ᴄ", 'd': "ᴅ", 'e': "ᴇ",
'f': "ғ", 'g': "ɢ", 'h': "ʜ", 'i': "ɪ", 'j': "ᴊ",
'k': "ᴋ", 'l': "ʟ", 'm': "ᴍ", 'n': "ɴ", 'o': "ᴏ",
'p': "ᴘ", 'q': "ǫ", 'r': "ʀ", 's': "s", 't': "ᴛ",
'u': "ᴜ", 'v': "ᴠ", 'w': "ᴡ", 'x': "x", 'y': "ʏ",
'z': "ᴢ",
}

func toFancy(s string) string {
var b strings.Builder
for _, r := range strings.ToLower(s) {
if mapped, ok := fancyMap[r]; ok {
b.WriteString(mapped)
} else {
b.WriteRune(r)
}
}
return b.String()
}

func cmdLines(cmds []*Command) string {
var sb strings.Builder
for _, cmd := range cmds {
sb.WriteString("│ ⟣ " + toFancy(cmd.Pattern) + "\n")
}
return sb.String()
}

func CategoryMenu(cat string) string {
cmds := categoryMap[strings.ToLower(cat)]
if len(cmds) == 0 {
return ""
}
var sb strings.Builder
sb.WriteString("╭─〔 *" + toFancy(cat) + "* MENU 〕\n")
sb.WriteString(cmdLines(cmds))
sb.WriteString("╰───────────────⬣")
return sb.String()
}

func formatUptime() string {
d := time.Since(botStartTime)
h := int(d.Hours())
m := int(d.Minutes()) % 60
return fmt.Sprintf("%dh %dm", h, m)
}

func getOS() string {
os := runtime.GOOS
switch os {
case "linux":
return "Linux"
case "windows":
return "Windows"
case "darwin":
return "MacOS"
default:
return os
}
}

func init() {
Register(&Command{
Pattern:  "menu",
Aliases:  []string{"help"},
Category: "utility",
Func: func(ctx *Context) error {
pushName := ctx.Event.Info.PushName
if pushName == "" {
pushName = ctx.Event.Info.Sender.User
}

totalCmds := len(registry)

var ms runtime.MemStats
runtime.ReadMemStats(&ms)
ramMB := ms.Alloc / 1024 / 1024

var sb strings.Builder
sb.WriteString("╭━━━〔 𝐙𝐀𝐄𝐋𝐈𝐗 〕━━━⬣\n")
sb.WriteString("┃◈ ᴜsᴇʀ      : " + pushName + "\n")
sb.WriteString("┃◈ ᴘʀᴇғɪx    : " + strings.Join(BotSettings.GetPrefixes(), " ") + "\n")
sb.WriteString("┃◈ ᴠᴇʀsɪᴏɴ   : v1.0.0\n")
sb.WriteString("┃◈ ᴜᴘᴛɪᴍᴇ    : " + formatUptime() + "\n")
sb.WriteString(fmt.Sprintf("┃◈ ᴘʟᴜɢɪɴs   : %d\n", totalCmds))
sb.WriteString(fmt.Sprintf("┃◈ ʀᴀᴍ       : %dMB\n", ramMB))
sb.WriteString("┃◈ ᴍᴏᴅᴇ      : " + string(BotSettings.GetMode()) + "\n")
sb.WriteString("┃◈ ʟᴀɴɢ      : " + BotSettings.GetLanguage() + "\n")
sb.WriteString("┃◈ ᴘʟᴀᴛғᴏʀᴍ  : " + getOS() + "\n")
sb.WriteString("╰━━━━━━━━━━━━━━━━━━⬣\n")

// Group by category
var catOrder []string
catMap := map[string][]*Command{}
for _, cmd := range registry {
cat := cmd.Category
if cat == "" {
cat = "general"
}
if _, exists := catMap[cat]; !exists {
catOrder = append(catOrder, cat)
}
catMap[cat] = append(catMap[cat], cmd)
}

catEmoji := map[string]string{
"settings": "🔧",
"ai":       "🤖",
"media":    "🎵",
"group":    "👥",
"download": "⬇️",
"utility":  "🛠️",
"general":  "📋",
}
for _, cat := range catOrder {
emoji := catEmoji[strings.ToLower(cat)]
if emoji == "" {
emoji = "•"
}
sb.WriteString("\n╭─━━〔 *" + emoji + " " + toFancy(cat) + "*〕\n")
sb.WriteString(cmdLines(catMap[cat]))
sb.WriteString("╰───────────────⬣\n")
}

ctx.Reply(strings.TrimRight(sb.String(), "\n"))
return nil
},
})
}
