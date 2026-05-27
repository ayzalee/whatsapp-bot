package main

import (
"context"
"flag"
"fmt"
"os"
"os/exec"
"os/signal"
"path/filepath"
"strings"
"syscall"
"time"

"zaelix/plugins"
"zaelix/store"
"zaelix/store/sqlstore"

"github.com/joho/godotenv"
"go.mau.fi/whatsmeow"
waProto "go.mau.fi/whatsmeow/proto/waE2E"
"go.mau.fi/whatsmeow/types"
waLog "go.mau.fi/whatsmeow/util/log"
"google.golang.org/protobuf/proto"

_ "github.com/lib/pq"
_ "modernc.org/sqlite"
)

var sourceDir string

func startSpinner(msg string) func(string) {
frames := []byte{'|', '/', '-', '\\'}
stop := make(chan string)
done := make(chan struct{})
go func() {
i := 0
t := time.NewTicker(80 * time.Millisecond)
defer t.Stop()
for {
select {
case m := <-stop:
fmt.Printf("\r%-70s\r%s\n", "", m)
close(done)
return
case <-t.C:
fmt.Printf("\r%c  %s", frames[i%len(frames)], msg)
i++
}
}
}()
return func(m string) { stop <- m; <-done }
}

func cliProgress(pct int, label string) {
const w = 28
filled := w * pct / 100
bar := strings.Repeat("━", filled) + strings.Repeat("─", w-filled)
if pct == 100 {
fmt.Printf("\r[%s] %3d%%  %-28s\n", bar, pct, label)
} else {
fmt.Printf("\r[%s] %3d%%  %-28s", bar, pct, label)
}
}

func printHelp() {
fmt.Print(`
 ╔══════════════════════════════════════════╗
 ║              Zaelix Bot                  ║
 ╚══════════════════════════════════════════╝

 Usage:
   zaelix [flags]

 Flags:
   --phone-number  <number>   Pair a new device
   --update                   Pull latest and rebuild
   --list-sessions            List all paired sessions
   --delete-session <number>  Delete a session
   --reset-session  <number>  Reset a session for re-pairing
   -h, --help                 Show this help

 Examples:
   zaelix                              Start bot
   zaelix --phone-number 923001234567  Pair device
   zaelix --update                     Update bot
   zaelix --list-sessions              Show sessions

`)
os.Exit(0)
}

func loadEnv() {
if err := godotenv.Load(".env"); err != nil {
_ = godotenv.Load(".env.example")
}
}

func dbConfig() (dialect, addr string) {
url := os.Getenv("DATABASE_URL")
if url == "" {
url = "database.db"
}
if strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://") {
return "postgres", url
}
path := strings.TrimPrefix(url, "file:")
addr = "file:" + path +
"?_pragma=foreign_keys(1)" +
"&_pragma=journal_mode(WAL)" +
"&_pragma=synchronous(NORMAL)" +
"&_pragma=busy_timeout(10000)" +
"&_pragma=cache_size(-64000)" +
"&_pragma=mmap_size(2147483648)" +
"&_pragma=temp_store(MEMORY)"
return "sqlite", addr
}

func getDevice(ctx context.Context, container *sqlstore.Container, phone string) (*store.Device, error) {
if phone == "" {
return container.GetFirstDevice(ctx)
}
devices, err := container.GetAllDevices(ctx)
if err != nil {
return nil, err
}
for _, dev := range devices {
if dev.ID == nil {
continue
}
if strings.SplitN(dev.ID.User, ".", 2)[0] == phone {
return dev, nil
}
}
return container.NewDevice(), nil
}

func boolEmoji(b bool) string {
if b {
return "✅"
}
return "❌"
}

func sendStartMessage(client *whatsmeow.Client, ownerPhone string) {
time.Sleep(3 * time.Second)
jid, err := types.ParseJID(ownerPhone + "@s.whatsapp.net")
if err != nil {
return
}
prefix := strings.Join(plugins.BotSettings.GetPrefixes(), " ")
msg := fmt.Sprintf(
"```BOT STARTED``` 🎐\n\n"+
"`ᴍᴏᴅᴇ`         : %s\n"+
"`ᴘʀᴇғɪx`        : %s\n"+
"`ʟᴀɴɢᴜᴀɢᴇ`      : %s\n\n"+
"*ᴀʟᴡᴀʏs ᴏɴʟɪɴᴇ*       : %s\n"+
"*ᴀᴜᴛᴏ sᴛᴀᴛᴜs ᴠɪᴇᴡ*    : %s\n"+
"*ᴀɴᴛɪ ᴅᴇʟᴇᴛᴇ ᴍsɢs*    : %s\n"+
"*ᴀᴜᴛᴏ ʀᴇᴊᴇᴄᴛ ᴄᴀʟʟs*   : %s\n"+
"*ᴀᴜᴛᴏ ʀᴇᴀᴅ ᴍsɢs*       : %s",
string(plugins.BotSettings.GetMode()),
prefix,
plugins.BotSettings.GetLanguage(),
boolEmoji(plugins.BotSettings.AlwaysOnline),
boolEmoji(plugins.BotSettings.AutoStatusView),
boolEmoji(plugins.GetAntiDeleteEnabled()),
boolEmoji(plugins.BotSettings.CallReject),
boolEmoji(plugins.GetAutoReadEnabled()),
)
client.SendMessage(context.Background(), jid, &waProto.Message{
Conversation: proto.String(msg),
})
}

func main() {
loadEnv()

flag.Usage = printHelp
helpFlag := flag.Bool("help", false, "")
phoneArg := flag.String("phone-number", "", "")
updateFlag := flag.Bool("update", false, "")
listFlag := flag.Bool("list-sessions", false, "")
deleteFlag := flag.String("delete-session", "", "")
resetFlag := flag.String("reset-session", "", "")
flag.Parse()

if *helpFlag {
printHelp()
}

ctx := context.Background()

if *updateFlag {
runUpdate()
return
}

dialect, dbAddr := dbConfig()

if *listFlag {
runListSessions(ctx, dialect, dbAddr)
return
}
if *deleteFlag != "" {
runDeleteSession(ctx, dialect, dbAddr, *deleteFlag, false)
return
}
if *resetFlag != "" {
runDeleteSession(ctx, dialect, dbAddr, *resetFlag, true)
return
}

dbLog := waLog.Stdout("Database", "ERROR", true)
container, err := sqlstore.New(ctx, dialect, dbAddr, dbLog)
if err != nil {
panic(err)
}

if err := plugins.InitDB(container.DB()); err != nil {
panic(fmt.Errorf("settings db init: %w", err))
}

plugins.InitLIDStore(container.LIDMap, "")

deviceStore, err := getDevice(ctx, container, *phoneArg)
if err != nil {
panic(err)
}

clientLog := waLog.Stdout("Client", "ERROR", true)
client := whatsmeow.NewClient(deviceStore, clientLog)
client.UseRetryMessageStore = true
client.AddEventHandler(plugins.NewHandler(client))

if err := container.LIDMap.FillCache(ctx); err != nil {
fmt.Fprintf(os.Stderr, "warn: FillCache: %v\n", err)
}

plugins.InitSourceDir(sourceDir)
plugins.SetRestartFunc(func() {
client.Disconnect()
exe, _ := os.Executable()
exe, _ = filepath.EvalSymlinks(exe)
cmd := exec.Command(exe, os.Args[1:]...)
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
_ = cmd.Start()
os.Exit(0)
})

stopSpin := startSpinner("Connecting to WhatsApp...")
if err = client.Connect(); err != nil {
stopSpin("Connection failed.")
panic(err)
}
stopSpin("Connected.")

if client.Store.ID == nil {
if *phoneArg == "" {
fmt.Println("No session found. Run with --phone-number <number> to pair.")
return
}
fmt.Println("Waiting 10 seconds before generating pairing code...")
time.Sleep(10 * time.Second)
code, err := client.PairPhone(ctx, *phoneArg, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
if err != nil {
panic(err)
}
fmt.Printf("\n  Pairing code: %s\n\n", code)
fmt.Println("  Open WhatsApp → Settings → Linked Devices → Link a Device")
fmt.Println("  Tap 'Link with phone number' and enter the code above.")
} else {
ownerPhone := strings.SplitN(client.Store.ID.User, ".", 2)[0]
plugins.InitLIDStore(container.LIDMap, ownerPhone)
if err := plugins.InitSettings(ownerPhone); err != nil {
panic(fmt.Errorf("settings load: %w", err))
}
plugins.BootstrapOwnerSudoers()
		plugins.StartScheduler(container.DB(), client)
plugins.ApplyEnvDefaults()

if plugins.BotSettings.AlwaysOnline {
plugins.StartOnlineLoop(client)
}
if plugins.BotSettings.AutoStatusView {
plugins.SetAutoViewStatus(true)
}
if plugins.BotSettings.CallReject {
plugins.SetCallReject(true)
}
if plugins.BotSettings.AntiDelete {
plugins.SetAntiDeleteEnabled(true)
}
if plugins.BotSettings.AutoRead {
plugins.SetAutoReadEnabled(true)
}

go sendStartMessage(client, ownerPhone)
fmt.Println("Already logged in.")
}

c := make(chan os.Signal, 1)
signal.Notify(c, os.Interrupt, syscall.SIGTERM)
<-c
client.Disconnect()
}

func candidateSourceDirs() []string {
candidates := []string{"/opt/zaelix/src"}
if pd := os.Getenv("ProgramData"); pd != "" {
candidates = append([]string{filepath.Join(pd, "zaelix", "src")}, candidates...)
}
if pf := os.Getenv("ProgramFiles"); pf != "" {
candidates = append(candidates, filepath.Join(pf, "zaelix", "src"))
}
return candidates
}

func resolveSourceDir() string {
if sourceDir != "" {
return sourceDir
}
for _, dir := range candidateSourceDirs() {
if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
return dir
}
}
return ""
}

func runUpdate() {
src := resolveSourceDir()
if src == "" {
fmt.Fprintln(os.Stderr, "error: source directory not found. Please reinstall.")
os.Exit(1)
}
sourceDir = src

cliProgress(0, "Fetching latest changes...")
if err := exec.Command("git", "-C", sourceDir, "fetch", "origin", "--quiet").Run(); err != nil {
fmt.Fprintf(os.Stderr, "\ngit fetch failed: %v\n", err)
os.Exit(1)
}
cliProgress(15, "Fetch complete")

out, _ := exec.Command("git", "-C", sourceDir, "rev-list", "HEAD..FETCH_HEAD", "--count").Output()
if strings.TrimSpace(string(out)) == "0" {
cliProgress(100, "Already up to date.")
return
}

cliProgress(20, "Pulling changes...")
pull := exec.Command("git", "-C", sourceDir, "pull", "--ff-only")
pull.Stdout = os.Stdout
pull.Stderr = os.Stderr
if err := pull.Run(); err != nil {
fmt.Fprintf(os.Stderr, "\ngit pull failed: %v\n", err)
os.Exit(1)
}
cliProgress(45, "Changes pulled")

exePath, err := os.Executable()
if err != nil {
fmt.Fprintf(os.Stderr, "\ncould not determine executable path: %v\n", err)
os.Exit(1)
}
exePath, _ = filepath.EvalSymlinks(exePath)
tmpPath := exePath + ".new"
ldflags := fmt.Sprintf("-s -w -X main.sourceDir=%s", sourceDir)

cliProgress(50, "Building new binary...")
buildDone := make(chan error, 1)
go func() {
cmd := exec.Command("go", "build", "-ldflags", ldflags, "-trimpath", "-o", tmpPath, ".")
cmd.Dir = sourceDir
buildDone <- cmd.Run()
}()

ticker := time.NewTicker(500 * time.Millisecond)
pct := 52
var buildErr error
buildLoop:
for {
select {
case buildErr = <-buildDone:
ticker.Stop()
break buildLoop
case <-ticker.C:
if pct < 88 {
pct++
cliProgress(pct, "Building new binary...")
}
}
}

if buildErr != nil {
_ = os.Remove(tmpPath)
fmt.Fprintf(os.Stderr, "\nbuild failed: %v\n", buildErr)
os.Exit(1)
}
cliProgress(90, "Build complete")

if err := os.Rename(tmpPath, exePath); err != nil {
fmt.Fprintf(os.Stderr, "\ncould not replace binary: %v\n", err)
fmt.Fprintf(os.Stderr, "New binary at: %s\nRename manually: mv %s %s\n", tmpPath, tmpPath, exePath)
os.Exit(1)
}
cliProgress(100, "Zaelix updated successfully.")
}

func runListSessions(ctx context.Context, dialect, dbAddr string) {
dbLog := waLog.Stdout("Database", "ERROR", true)
container, err := sqlstore.New(ctx, dialect, dbAddr, dbLog)
if err != nil {
fmt.Fprintf(os.Stderr, "Failed to open database: %v\n", err)
os.Exit(1)
}
devices, err := container.GetAllDevices(ctx)
if err != nil {
fmt.Fprintf(os.Stderr, "Failed to list sessions: %v\n", err)
os.Exit(1)
}
if len(devices) == 0 {
fmt.Println("No sessions found.")
return
}
fmt.Printf("%-4s  %-20s  %s\n", "No.", "Phone", "JID")
fmt.Println(strings.Repeat("-", 60))
for i, dev := range devices {
phone, jid := "(unknown)", "(unpaired)"
if dev.ID != nil {
phone = strings.SplitN(dev.ID.User, ".", 2)[0]
jid = dev.ID.String()
}
fmt.Printf("%-4d  %-20s  %s\n", i+1, phone, jid)
}
}

func runDeleteSession(ctx context.Context, dialect, dbAddr, phone string, reset bool) {
dbLog := waLog.Stdout("Database", "ERROR", true)
container, err := sqlstore.New(ctx, dialect, dbAddr, dbLog)
if err != nil {
fmt.Fprintf(os.Stderr, "Failed to open database: %v\n", err)
os.Exit(1)
}
devices, err := container.GetAllDevices(ctx)
if err != nil {
fmt.Fprintf(os.Stderr, "Failed to query sessions: %v\n", err)
os.Exit(1)
}
for _, dev := range devices {
if dev.ID == nil {
continue
}
if strings.SplitN(dev.ID.User, ".", 2)[0] == phone {
if err := container.DeleteDevice(ctx, dev); err != nil {
fmt.Fprintf(os.Stderr, "Failed to delete session: %v\n", err)
os.Exit(1)
}
if reset {
fmt.Printf("Session for %s reset. Run --phone-number %s to re-pair.\n", phone, phone)
} else {
fmt.Printf("Session for %s permanently deleted.\n", phone)
}
return
}
}
fmt.Fprintf(os.Stderr, "No session found for: %s\n", phone)
os.Exit(1)
}
