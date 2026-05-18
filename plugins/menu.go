package plugins

import (
"fmt"
"strings"
)

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
line := toFancy(cmd.Pattern)
if len(cmd.Aliases) > 0 {
parts := make([]string, len(cmd.Aliases))
for i, a := range cmd.Aliases {
parts[i] = toFancy(a)
}
line += "  [" + strings.Join(parts, ", ") + "]"
}
sb.WriteString("  · " + line + "\n")
}
return sb.String()
}

func CategoryMenu(cat string) string {
cmds := categoryMap[strings.ToLower(cat)]
if len(cmds) == 0 {
return ""
}
return "*" + toFancy(cat) + " ᴍᴇɴᴜ*\n\n" + strings.TrimRight(cmdLines(cmds), "\n")
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

var sb strings.Builder
fmt.Fprintf(&sb, T().MenuGreeting+"\n", pushName)

for _, cat := range catOrder {
sb.WriteString("\n▸ *" + toFancy(cat) + "*\n")
sb.WriteString(cmdLines(catMap[cat]))
}

sb.WriteString("\n\n> © ᴀʏᴀᴢᴀʟɪᴏꜰᴄ")

ctx.Reply(strings.TrimRight(sb.String(), "\n"))
return nil
},
})
}
