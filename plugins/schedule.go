package plugins

import (
"database/sql"
"fmt"
"strconv"
"strings"
"sync"
"time"

"go.mau.fi/whatsmeow"
waProto "go.mau.fi/whatsmeow/proto/waE2E"
"go.mau.fi/whatsmeow/types"
"google.golang.org/protobuf/proto"
)

type Schedule struct {
ID      int64
JID     string
Message string
NextRun time.Time
Repeat  bool
Cron    string // "HH:MM" or "daily HH:MM" or "weekly N HH:MM"
}

var (
scheduleMu     sync.Mutex
scheduleClient interface {
SendMessage(ctx interface{}, jid types.JID, msg *waProto.Message, extra ...interface{}) (interface{}, error)
GenerateMessageID() string
}
)

func initScheduleDB(db *sql.DB) error {
_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schedules (
id INTEGER PRIMARY KEY AUTOINCREMENT,
jid TEXT NOT NULL,
message TEXT NOT NULL,
next_run INTEGER NOT NULL,
repeat INTEGER NOT NULL DEFAULT 0,
cron TEXT NOT NULL DEFAULT ''
)`)
return err
}

func loadSchedules(db *sql.DB) ([]*Schedule, error) {
rows, err := db.Query(`SELECT id, jid, message, next_run, repeat, cron FROM schedules`)
if err != nil {
return nil, err
}
defer rows.Close()
var schedules []*Schedule
for rows.Next() {
s := &Schedule{}
var nextRun int64
if err := rows.Scan(&s.ID, &s.JID, &s.Message, &nextRun, &s.Repeat, &s.Cron); err != nil {
continue
}
s.NextRun = time.Unix(nextRun, 0)
schedules = append(schedules, s)
}
return schedules, nil
}

func saveSchedule(db *sql.DB, s *Schedule) (int64, error) {
res, err := db.Exec(
`INSERT INTO schedules (jid, message, next_run, repeat, cron) VALUES (?, ?, ?, ?, ?)`,
s.JID, s.Message, s.NextRun.Unix(), boolToInt(s.Repeat), s.Cron,
)
if err != nil {
return 0, err
}
return res.LastInsertId()
}

func deleteSchedule(db *sql.DB, id int64) error {
_, err := db.Exec(`DELETE FROM schedules WHERE id = ?`, id)
return err
}

func updateScheduleNextRun(db *sql.DB, id int64, next time.Time) error {
_, err := db.Exec(`UPDATE schedules SET next_run = ? WHERE id = ?`, next.Unix(), id)
return err
}

func boolToInt(b bool) int {
if b {
return 1
}
return 0
}

func parseScheduleTime(arg string) (nextRun time.Time, repeat bool, cron string, msg string, err error) {
now := time.Now()

if strings.HasPrefix(strings.ToLower(arg), "every day ") {
rest := arg[len("every day "):]
parts := strings.SplitN(rest, " ", 2)
if len(parts) < 2 {
err = fmt.Errorf("usage: every day HH:MM <message>")
return
}
t, e := parseHHMM(parts[0])
if e != nil {
err = e
return
}
nextRun = nextOccurrence(now, t.Hour(), t.Minute())
repeat = true
cron = "daily " + parts[0]
msg = parts[1]
return
}

if strings.HasPrefix(strings.ToLower(arg), "every week ") {
rest := arg[len("every week "):]
parts := strings.SplitN(rest, " ", 3)
if len(parts) < 3 {
err = fmt.Errorf("usage: every week <weekday> HH:MM <message>")
return
}
weekday, e := parseWeekday(parts[0])
if e != nil {
err = e
return
}
t, e := parseHHMM(parts[1])
if e != nil {
err = e
return
}
nextRun = nextWeekdayOccurrence(now, weekday, t.Hour(), t.Minute())
repeat = true
cron = fmt.Sprintf("weekly %d %s", weekday, parts[1])
msg = parts[2]
return
}

if strings.ToLower(arg) == "tomorrow" || strings.HasPrefix(strings.ToLower(arg), "tomorrow ") {
rest := strings.TrimPrefix(strings.TrimPrefix(arg, "tomorrow "), "tomorrow")
parts := strings.SplitN(strings.TrimSpace(rest), " ", 2)
if len(parts) < 2 {
err = fmt.Errorf("usage: tomorrow HH:MM <message>")
return
}
t, e := parseHHMM(parts[0])
if e != nil {
err = e
return
}
tomorrow := now.AddDate(0, 0, 1)
nextRun = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
repeat = false
cron = ""
msg = parts[1]
return
}

// Default: HH:MM <message>
parts := strings.SplitN(arg, " ", 2)
if len(parts) < 2 {
err = fmt.Errorf("usage: HH:MM <message>")
return
}
t, e := parseHHMM(parts[0])
if e != nil {
err = e
return
}
nextRun = nextOccurrence(now, t.Hour(), t.Minute())
repeat = false
cron = ""
msg = parts[1]
return
}

func parseHHMM(s string) (time.Time, error) {
parts := strings.Split(s, ":")
if len(parts) != 2 {
return time.Time{}, fmt.Errorf("invalid time format, use HH:MM")
}
h, err := strconv.Atoi(parts[0])
if err != nil || h < 0 || h > 23 {
return time.Time{}, fmt.Errorf("invalid hour")
}
m, err := strconv.Atoi(parts[1])
if err != nil || m < 0 || m > 59 {
return time.Time{}, fmt.Errorf("invalid minute")
}
return time.Date(0, 1, 1, h, m, 0, 0, time.Local), nil
}

func parseWeekday(s string) (time.Weekday, error) {
days := map[string]time.Weekday{
"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
"wednesday": time.Wednesday, "thursday": time.Thursday, "friday": time.Friday,
"saturday": time.Saturday,
}
if d, ok := days[strings.ToLower(s)]; ok {
return d, nil
}
return 0, fmt.Errorf("invalid weekday: %s", s)
}

func nextOccurrence(now time.Time, hour, minute int) time.Time {
t := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.Local)
if !t.After(now) {
t = t.AddDate(0, 0, 1)
}
return t
}

func nextWeekdayOccurrence(now time.Time, weekday time.Weekday, hour, minute int) time.Time {
t := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.Local)
for t.Weekday() != weekday || !t.After(now) {
t = t.AddDate(0, 0, 1)
}
return t
}

func nextRunFromCron(cron string) time.Time {
now := time.Now()
parts := strings.Fields(cron)
if len(parts) == 0 {
return time.Time{}
}
switch parts[0] {
case "daily":
if len(parts) < 2 {
return time.Time{}
}
t, err := parseHHMM(parts[1])
if err != nil {
return time.Time{}
}
return nextOccurrence(now, t.Hour(), t.Minute())
case "weekly":
if len(parts) < 3 {
return time.Time{}
}
day, _ := strconv.Atoi(parts[1])
t, err := parseHHMM(parts[2])
if err != nil {
return time.Time{}
}
return nextWeekdayOccurrence(now, time.Weekday(day), t.Hour(), t.Minute())
}
return time.Time{}
}

var scheduleDB *sql.DB

func StartScheduler(db *sql.DB, client interface{}) {
scheduleDB = db
if err := initScheduleDB(db); err != nil {
return
}
go func() {
ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()
for range ticker.C {
runDueSchedules(client)
}
}()
}

func runDueSchedules(client interface{}) {
if scheduleDB == nil {
return
}
scheduleMu.Lock()
	wc, ok := client.(*whatsmeow.Client)
	if !ok {
		return
	}
defer scheduleMu.Unlock()

schedules, err := loadSchedules(scheduleDB)
if err != nil {
return
}

now := time.Now()
for _, s := range schedules {
if now.Before(s.NextRun) {
continue
}

jid, err := types.ParseJID(s.JID)
if err != nil {
deleteSchedule(scheduleDB, s.ID)
continue
}

sendQueue <- sendTask{
		client: wc,
to:     jid,
msg:    &waProto.Message{Conversation: proto.String(s.Message)},
		id:     wc.GenerateMessageID(),
}

if s.Repeat && s.Cron != "" {
next := nextRunFromCron(s.Cron)
if !next.IsZero() {
updateScheduleNextRun(scheduleDB, s.ID, next)
} else {
deleteSchedule(scheduleDB, s.ID)
}
} else {
deleteSchedule(scheduleDB, s.ID)
}
}
}

func init() {
Register(&Command{
Pattern:  "setschedule",
IsSudo:   true,
Category: "utility",
Func: func(ctx *Context) error {
arg := strings.TrimSpace(ctx.Text)
if arg == "" {
ctx.Reply("> *Schedule Usage:*\n\n" +
"*.setschedule <jid> HH:MM <message>* — once\n" +
"*.setschedule <jid> every day HH:MM <message>* — daily\n" +
"*.setschedule <jid> every week <weekday> HH:MM <message>* — weekly\n" +
"*.setschedule <jid> tomorrow HH:MM <message>* — tomorrow\n\n" +
"_JID example: 923001234567 or 923001234567@s.whatsapp.net_")
return nil
}

parts := strings.SplitN(arg, " ", 2)
if len(parts) < 2 {
ctx.Reply(T().SchedUsage)
return nil
}

jidStr := parts[0]
if !strings.Contains(jidStr, "@") {
if len(jidStr) > 15 {
jidStr += "@g.us"
} else {
jidStr += "@s.whatsapp.net"
}
}

if _, err := types.ParseJID(jidStr); err != nil {
ctx.Reply(fmt.Sprintf(T().SchedInvalidJID, jidStr))
return nil
}

nextRun, repeat, cron, msg, err := parseScheduleTime(parts[1])
if err != nil {
ctx.Reply("Error: " + err.Error())
return nil
}

if msg == "" {
ctx.Reply(T().SchedEmptyMsg)
return nil
}

s := &Schedule{
JID:     jidStr,
Message: msg,
NextRun: nextRun,
Repeat:  repeat,
Cron:    cron,
}

id, err := saveSchedule(scheduleDB, s)
if err != nil {
ctx.Reply(fmt.Sprintf(T().SchedSaveFailed, err.Error()))
return nil
}

repeatStr := "once"
if repeat {
repeatStr = "repeating (" + cron + ")"
}
ctx.Reply(fmt.Sprintf(T().SchedCreated,
id, jidStr, nextRun.Format("02 Jan 2006 15:04"), repeatStr, msg))
return nil
},
})

Register(&Command{
Pattern:  "getschedule",
IsSudo:   true,
Category: "utility",
Func: func(ctx *Context) error {
if scheduleDB == nil {
ctx.Reply(T().SchedNotInit)
return nil
}
schedules, err := loadSchedules(scheduleDB)
if err != nil || len(schedules) == 0 {
ctx.Reply(T().SchedEmpty)
return nil
}
var sb strings.Builder
sb.WriteString("*Scheduled Messages:*\n\n")
for _, s := range schedules {
repeatStr := "once"
if s.Repeat {
repeatStr = "repeating"
}
sb.WriteString(fmt.Sprintf("*#%d* → %s\n📅 %s | %s\n💬 %s\n\n",
s.ID, s.JID, s.NextRun.Format("02 Jan 15:04"), repeatStr, s.Message))
}
ctx.Reply(strings.TrimRight(sb.String(), "\n"))
return nil
},
})

Register(&Command{
Pattern:  "delschedule",
IsSudo:   true,
Category: "utility",
Func: func(ctx *Context) error {
arg := strings.TrimSpace(ctx.Text)
if arg == "" {
ctx.Reply(T().SchedDelUsage)
return nil
}
if arg == "all" {
scheduleDB.Exec(`DELETE FROM schedules`)
ctx.Reply(T().SchedDelAll)
return nil
}
id, err := strconv.ParseInt(arg, 10, 64)
if err != nil {
ctx.Reply(T().SchedDelInvalid)
return nil
}
if err := deleteSchedule(scheduleDB, id); err != nil {
ctx.Reply(fmt.Sprintf(T().SchedDelFailed, err.Error()))
return nil
}
ctx.Reply(fmt.Sprintf(T().SchedDeleted, id))
return nil
},
})
}
