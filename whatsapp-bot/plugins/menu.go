package plugins

import (
	"fmt"
	"strings"
)

// fancyMap maps ASCII letters and digits to small-caps / monospace Unicode equivalents.
// Digit order in the source string: 1234567890  →  𝟷𝟸𝟹𝟺𝟻𝟼𝟽𝟾𝟿𝟶
// Letter order: abcdefghijklmnopqrstuvwxyz → ᴀʙᴄᴅᴇғɢʜɪᴊᴋʟᴍɴᴏᴘǫʀsᴛᴜᴠᴡxʏᴢ
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

// toFancy converts a string to fancy small-caps Unicode text.
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

// cmdLines renders a slice of commands as bulleted lines.
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

// CategoryMenu builds a message listing commands for a single category.
// Returns an empty string when no commands belong to that category.
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

			// Group commands by category, preserving first-seen order.
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

			ctx.Reply(strings.TrimRight(sb.String(), "\n"))
			return nil
		},
	})
}

