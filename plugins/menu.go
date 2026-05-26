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
		line := toFancy(cmd.Pattern)
		if len(cmd.Aliases) > 0 {
			parts := make([]string, len(cmd.Aliases))
			for i, a := range cmd.Aliases {
				parts[i] = toFancy(a)
			}
			line += "  [" + strings.Join(parts, ", ") + "]"
		}
		sb.WriteString("│ ◈ " + line + "\n")
	}
	return sb.String()
}

func CategoryMenu(cat string) string {
	cmds := categoryMap[strings.ToLower(cat)]
	if len(cmds) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("╭─〔 *✦ " + toFancy(cat) + " ✦* 〕\n")
	sb.WriteString(cmdLines(cmds))
	sb.WriteString("╰────────────────⊷")
	return sb.String()
}

func formatUptime() string {
	d := time.Since(botStartTime)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dʜ %dᴍ %ds", h, m, s)
}

func getRamMB() uint64 {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return ms.Alloc / 1024 / 1024
}

func getOS() string {
	switch runtime.GOOS {
	case "linux":
		return "ᴠᴘs (Linux)"
	case "darwin":
		return "ᴍᴀᴄᴏs"
	case "android":
		return "ᴀɴᴅʀᴏɪᴅ"
	default:
		return runtime.GOOS
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

			now := time.Now()
			totalCmds := len(registry)

			var sb strings.Builder

			prefix := strings.Join(BotSettings.GetPrefixes(), " ")
			sb.WriteString("╭═══〔 𝐙ᴀᴇʟɪx 〕═══⊷\n")
			sb.WriteString("┃❒╭──────────────\n")
			sb.WriteString("┃❒│ *ᴘʀᴇғɪx*   : `" + prefix + "`\n")
			sb.WriteString("┃❒│ *ᴜsᴇʀ*     : `" + pushName + "`\n")
			sb.WriteString("┃❒│ *ᴛɪᴍᴇ*     : `" + now.Format("03:04 PM") + "`\n")
			sb.WriteString("┃❒│ *ᴅᴀʏ*      : `" + toFancy(now.Weekday().String()) + "`\n")
			sb.WriteString("┃❒│ *ᴅᴀᴛᴇ*     : `" + now.Format("02/01/2006") + "`\n")
			sb.WriteString(fmt.Sprintf("┃❒│ *ᴘʟᴜɢɪɴs*  : `%d`\n", totalCmds))
			sb.WriteString("┃❒│ *ᴜᴘᴛɪᴍᴇ*   : `" + formatUptime() + "`\n")
			sb.WriteString("┃❒│ *ᴍᴏᴅᴇ*     : `" + toFancy(string(BotSettings.GetMode())) + "`\n")
			sb.WriteString("┃❒│ *ᴘʟᴀᴛғᴏʀᴍ* : `" + getOS() + "`\n")
			sb.WriteString("┃❒╰──────────────\n")
			sb.WriteString("╰═════════════════⊷\n")

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

			for _, cat := range catOrder {
				sb.WriteString("\n╭─〔 *✦ " + toFancy(cat) + " ✦* 〕\n")
				sb.WriteString(cmdLines(catMap[cat]))
				sb.WriteString("╰────────────────⊷\n")
			}

			ctx.Reply(strings.TrimRight(sb.String(), "\n"))
			return nil
		},
	})
}
