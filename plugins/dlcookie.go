package plugins

import (
"os"
"strings"
)

var dlCookieFile string

func InitDLCookie() {
if cookieStr := os.Getenv("YT_COOKIE"); cookieStr != "" {
if err := saveCookieFile(cookieStr); err == nil {
dlCookieFile = "cookies.txt"
}
}
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
