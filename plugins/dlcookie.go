package plugins

import (
"os"
"strings"
)

var dlCookieFile string

func init() {
// Load cookie file from env on startup
if cookieStr := os.Getenv("YT_COOKIE"); cookieStr != "" {
if err := saveCookieFile(cookieStr); err == nil {
dlCookieFile = "cookies.txt"
}
}

Register(&Command{
Pattern:  "dlcookie",
IsSudo:   true,
Category: "download",
Func: func(ctx *Context) error {
arg := strings.TrimSpace(ctx.Text)
if arg == "" {
if dlCookieFile != "" {
ctx.Reply("Cookie is set.\n\n.dlcookie clear — remove")
} else {
ctx.Reply("No cookie set.\n\nUsage: .dlcookie <cookie_string>")
}
return nil
}
if arg == "clear" {
dlCookieFile = ""
os.Remove("cookies.txt")
ctx.Reply("Cookie cleared.")
return nil
}
if err := saveCookieFile(arg); err != nil {
ctx.Reply("Failed to save cookie: " + err.Error())
return nil
}
dlCookieFile = "cookies.txt"
ctx.Reply("Cookie saved.")
return nil
},
})
}

func saveCookieFile(cookieStr string) error {
lines := "# Netscape HTTP Cookie File\n"
for _, pair := range strings.Split(cookieStr, ";") {
pair = strings.TrimSpace(pair)
if pair == "" {
continue
}
parts := strings.SplitN(pair, "=", 2)
if len(parts) != 2 {
continue
}
name := strings.TrimSpace(parts[0])
value := strings.TrimSpace(parts[1])
lines += ".youtube.com\tTRUE\t/\tTRUE\t2147483647\t" + name + "\t" + value + "\n"
}
return os.WriteFile("cookies.txt", []byte(lines), 0644)
}
