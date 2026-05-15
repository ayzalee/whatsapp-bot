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
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

// sourceDir is injected at build time via:
//
//	-ldflags "-X main.sourceDir=/path/to/src"
var sourceDir string

// ── CLI presentation helpers ──────────────────────────────────────────────────

// startSpinner prints a rotating spinner with msg until the returned stop
// function is called. stop(done) clears the line and prints done.
func startSpinner(msg string) func(done string) {
	frames := []byte{'|', '/', '-', '\\'}
	stop := make(chan string) // unbuffered — send blocks until goroutine receives
	finished := make(chan struct{})
	go func() {
		i := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case doneMsg := <-stop:
				fmt.Printf("\r%-70s\r%s\n", "", doneMsg)
				close(finished)
				return
			case <-ticker.C:
				fmt.Printf("\r%c  %s", frames[i%len(frames)], msg)
				i++
			}
		}
	}()
	return func(doneMsg string) {
		stop <- doneMsg // blocks until goroutine receives and clears the line
		<-finished      // blocks until done message is printed
	}
}

// cliProgress prints an in-place progress bar.
// When pct == 100 it prints a newline to finalise the line.
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

// printHelp prints a formatted usage/help screen and exits.
func printHelp() {
	fmt.Print(`
 ╔══════════════════════════════════════════╗
 ║           Ortsax Menu                    ║
 ╚══════════════════════════════════════════╝

 Usage
   zaelix [flags]

 Flags
   --phone-number  <number>   Phone number (international format) to
                              identify or pair a device
   --update                   Pull latest source and rebuild binary
   --list-sessions            List all paired sessions in the database
   --delete-session <number>  Permanently delete a session by phone
   --reset-session  <number>  Reset a session so it can be re-paired
   -h, --help                 Show this help screen

 Examples
   zaelix                           Start the bot (uses stored session)
   zaelix --phone-number 2348000000 Pair a new device
   zaelix --update                  Update to latest version
   zaelix --list-sessions           Show all saved sessions

`)
	os.Exit(0)
}

func loadEnv() {
	if err := godotenv.Load(".env"); err != nil {
		_ = godotenv.Load(".env.example")
	}
}

// dbConfig returns the sql dialect and connection address derived from DATABASE_URL.
// A bare filename (no scheme) or a path ending in .db is treated as SQLite;
// anything starting with postgres:// or postgresql:// is treated as PostgreSQL.
func dbConfig() (dialect, addr string) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "database.db"
	}

	if strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://") {
		return "postgres", url
	}

	// SQLite – build the connection string with recommended pragmas.
	// Strip a leading "file:" if present so we can normalise the path.
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

// getDevice returns the device for the given phone number.
// If phone is empty it falls back to the first stored device (or a new one).
// If phone is provided and no matching device exists, a new (unpaired) device is returned.
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
		// Device JID User field may be "phone.deviceIndex" – compare only the phone part.
		userPhone := strings.SplitN(dev.ID.User, ".", 2)[0]
		if userPhone == phone {
			return dev, nil
		}
	}
	// No existing session for this number – return a fresh device for pairing.
	return container.NewDevice(), nil
}

func main() {
	loadEnv()

	// ── CLI flags ────────────────────────────────────────────────────────────
	flag.Usage = printHelp
	helpFlag := flag.Bool("help", false, "")
	phoneArg := flag.String("phone-number", "", "Phone number (international format) used to identify or pair a device")
	updateFlag := flag.Bool("update", false, "Pull latest source and rebuild the binary in-place")
	listFlag := flag.Bool("list-sessions", false, "List all paired sessions stored in the database")
	deleteFlag := flag.String("delete-session", "", "Permanently delete the session for the given phone number")
	resetFlag := flag.String("reset-session", "", "Reset the session for the given phone number so it can be re-paired")
	flag.Parse()

	if *helpFlag {
		printHelp()
	}

	ctx := context.Background()

	// ── Management commands (exit after completion) ───────────────────────────
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

	// ── Normal bot startup ────────────────────────────────────────────────────
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

	// Pre-warm LID↔PN in-memory cache so DM sends don't pay per-lookup
	// SQLite overhead on first contact.
	if err := container.LIDMap.FillCache(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "warn: FillCache: %v\n", err)
	}

	// Wire update command with source dir and restart capability.
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
	err = client.Connect()
	if err != nil {
		stopSpin("Connection failed.")
		panic(err)
	}
	stopSpin("Connected.")

	if client.Store.ID == nil {
		if *phoneArg == "" {
			fmt.Println("No session found. Please provide a phone number using --phone-number")
			return
		}

		fmt.Println("Waiting 10 seconds before generating pairing code...")
		time.Sleep(10 * time.Second)

		code, err := client.PairPhone(ctx, *phoneArg, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
		if err != nil {
			panic(err)
		}
		fmt.Printf("Your pairing code is: %s\n", code)
	} else {
		ownerPhone := strings.SplitN(client.Store.ID.User, ".", 2)[0]
		plugins.InitLIDStore(container.LIDMap, ownerPhone)
		if err := plugins.InitSettings(ownerPhone); err != nil {
			panic(fmt.Errorf("settings load: %w", err))
		}
		plugins.BootstrapOwnerSudoers()
			if plugins.BotSettings.AlwaysOnline {
				plugins.StartOnlineLoop(client)
			}
		fmt.Println("Already logged in.")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}

// ── Management command handlers ───────────────────────────────────────────────

// candidateSourceDirs returns the standard install-time source directories
// to check when the binary was not built with -X main.sourceDir.
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

// resolveSourceDir returns sourceDir if set, otherwise searches well-known
// install locations for a valid git repository.
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

// runUpdate pulls the latest source and rebuilds the binary in-place.
func runUpdate() {
	src := resolveSourceDir()
	if src == "" {
		fmt.Fprintln(os.Stderr, "error: could not locate the zaelix source directory.")
		fmt.Fprintln(os.Stderr, "Please reinstall using the install script.")
		os.Exit(1)
	}
	sourceDir = src

	cliProgress(0, "Fetching latest changes...")
	fetch := exec.Command("git", "-C", sourceDir, "fetch", "origin", "--quiet")
	fetch.Stderr = os.Stderr
	if err := fetch.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\ngit fetch failed: %v\n", err)
		os.Exit(1)
	}
	cliProgress(15, "Fetch complete")

	// Check if there is anything to pull.
	countOut, _ := exec.Command("git", "-C", sourceDir, "rev-list", "HEAD..FETCH_HEAD", "--count").Output()
	if strings.TrimSpace(string(countOut)) == "0" {
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
		cmd := exec.Command("go", "build",
			"-ldflags", ldflags,
			"-trimpath",
			"-o", tmpPath,
			".",
		)
		cmd.Dir = sourceDir
		buildDone <- cmd.Run()
	}()

	// Animate 52→88% while build runs (tick every 500ms).
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
		fmt.Fprintf(os.Stderr, "\ncould not replace binary (stop the bot first if it is running): %v\n", err)
		fmt.Fprintf(os.Stderr, "New binary saved at: %s\nRename manually: mv %s %s\n", tmpPath, tmpPath, exePath)
		os.Exit(1)
	}
	cliProgress(100, "Ortsax updated successfully.")
}

// runListSessions opens the database and prints all paired sessions.
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
		phone := "(unknown)"
		jid := "(unpaired)"
		if dev.ID != nil {
			phone = strings.SplitN(dev.ID.User, ".", 2)[0]
			jid = dev.ID.String()
		}
		fmt.Printf("%-4d  %-20s  %s\n", i+1, phone, jid)
	}
}

// runDeleteSession removes the stored session for the given phone number.
// When reset is true the message instructs the user to re-pair.
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
				fmt.Printf("Session for %s has been reset.\nRun with --phone-number %s to re-pair.\n", phone, phone)
			} else {
				fmt.Printf("Session for %s has been permanently deleted.\n", phone)
			}
			return
		}
	}

	fmt.Fprintf(os.Stderr, "No session found for phone number: %s\n", phone)
	os.Exit(1)
}
