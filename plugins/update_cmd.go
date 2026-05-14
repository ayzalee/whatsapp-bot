package plugins

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

var botSourceDir string
var restartFunc func()

// InitSourceDir sets the source directory used by the update command.
// Call this from main after loading env.
func InitSourceDir(dir string) { botSourceDir = dir }

// SetRestartFunc registers the function that disconnects and re-execs the process.
func SetRestartFunc(f func()) { restartFunc = f }

const updateBarWidth = 22

// updateBar renders a WhatsApp-formatted progress bar.
// The filled portion is wrapped in *…* so WhatsApp renders it bold.
//
//	0%  : ──────────────────────  0%
//	50% : *━━━━━━━━━━━*───────────  50%
//	100%: *━━━━━━━━━━━━━━━━━━━━━━* 100%
func updateBar(pct int) string {
	filled := updateBarWidth * pct / 100
	var sb strings.Builder
	if filled > 0 {
		sb.WriteString("*")
		for range filled {
			sb.WriteRune('━')
		}
		sb.WriteString("*")
	}
	for i := filled; i < updateBarWidth; i++ {
		sb.WriteRune('─')
	}
	fmt.Fprintf(&sb, "  %d%%", pct)
	return sb.String()
}

// editUpdate edits msgID in chatJID with a formatted update status message.
func editUpdate(ctx *Context, chatJID types.JID, msgID, label string, pct int) {
	text := fmt.Sprintf("Updating Ortsax...\n%s\n%s", updateBar(pct), label)
	edit := ctx.Client.BuildEdit(chatJID, msgID, &waProto.Message{
		Conversation: proto.String(text),
	})
	ctx.Client.SendMessage(context.Background(), chatJID, edit)
	// Small pause to avoid WhatsApp edit rate-limiting.
	time.Sleep(250 * time.Millisecond)
}

// gitRun runs a git command inside botSourceDir.
func gitRun(args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", botSourceDir}, args...)...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// pendingCommits returns how many commits origin is ahead of HEAD.
func pendingCommits() (int, error) {
	if err := gitRun("fetch", "origin", "--quiet"); err != nil {
		return 0, err
	}
	out, err := exec.Command("git", "-C", botSourceDir, "rev-list", "HEAD..FETCH_HEAD", "--count").Output()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(out)))
}

func init() {
	Register(&Command{
		Pattern:  "update",
		IsSudo:   true,
		Category: "utility",
		Func: func(ctx *Context) error {
			if botSourceDir == "" {
				ctx.Reply("Update not available: source directory not embedded in this binary.\nReinstall using the install script.")
				return nil
			}

			mode := strings.ToLower(strings.TrimSpace(ctx.Text))

			// ── .update — check only ──────────────────────────────────────────
			if mode == "" {
				resp, err := ctx.ReplySync("Checking for updates...")
				if err != nil {
					return err
				}
				n, err := pendingCommits()
				if err != nil {
					edit := ctx.Client.BuildEdit(ctx.Event.Info.Chat, resp.ID, &waProto.Message{
						Conversation: proto.String("Failed to check for updates:\n" + err.Error()),
					})
					ctx.Client.SendMessage(context.Background(), ctx.Event.Info.Chat, edit)
					return nil
				}
				var msg string
				if n == 0 {
					msg = "Already up to date."
				} else {
					msg = fmt.Sprintf("%d new commit(s) available.\nUse *update now* to apply.", n)
				}
				edit := ctx.Client.BuildEdit(ctx.Event.Info.Chat, resp.ID, &waProto.Message{
					Conversation: proto.String(msg),
				})
				ctx.Client.SendMessage(context.Background(), ctx.Event.Info.Chat, edit)
				return nil
			}

			// ── .update now — apply update ────────────────────────────────────
			if mode != "now" {
				ctx.Reply("Usage:\n  update       — check for updates\n  update now   — download and apply updates")
				return nil
			}

			chatJID := ctx.Event.Info.Chat
			resp, err := ctx.ReplySync("Updating Ortsax...\n" + updateBar(0) + "\n  Starting...")
			if err != nil {
				return err
			}
			msgID := resp.ID

			// Step 1 — fetch
			editUpdate(ctx, chatJID, msgID, "Fetching latest changes...", 5)
			if err := gitRun("fetch", "origin", "--quiet"); err != nil {
				editUpdate(ctx, chatJID, msgID, "Failed to fetch: "+err.Error(), 5)
				return nil
			}
			editUpdate(ctx, chatJID, msgID, "Fetch complete", 15)

			// Check if pull is needed
			out, _ := exec.Command("git", "-C", botSourceDir, "rev-list", "HEAD..FETCH_HEAD", "--count").Output()
			if strings.TrimSpace(string(out)) == "0" {
				text := fmt.Sprintf("Already up to date.\n%s", updateBar(100))
				edit := ctx.Client.BuildEdit(chatJID, msgID, &waProto.Message{Conversation: proto.String(text)})
				ctx.Client.SendMessage(context.Background(), chatJID, edit)
				return nil
			}

			// Step 2 — pull
			editUpdate(ctx, chatJID, msgID, "Pulling changes...", 20)
			if err := gitRun("pull", "--ff-only"); err != nil {
				editUpdate(ctx, chatJID, msgID, "Pull failed: "+err.Error(), 20)
				return nil
			}
			editUpdate(ctx, chatJID, msgID, "Changes pulled", 45)

			// Step 3 — build
			editUpdate(ctx, chatJID, msgID, "Building new binary...", 50)

			exePath, err := os.Executable()
			if err != nil {
				editUpdate(ctx, chatJID, msgID, "Cannot resolve executable path: "+err.Error(), 50)
				return nil
			}
			exePath, _ = filepath.EvalSymlinks(exePath)
			tmpPath := exePath + ".new"
			ldflags := fmt.Sprintf("-s -w -X main.sourceDir=%s", botSourceDir)

			buildDone := make(chan error, 1)
			go func() {
				cmd := exec.Command("go", "build",
					"-ldflags", ldflags,
					"-trimpath",
					"-o", tmpPath,
					".",
				)
				cmd.Dir = botSourceDir
				buildDone <- cmd.Run()
			}()

			// Animate 52→88% while build runs.
			ticker := time.NewTicker(2 * time.Second)
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
						pct += 3
						editUpdate(ctx, chatJID, msgID, "Building new binary...", pct)
					}
				}
			}

			if buildErr != nil {
				_ = os.Remove(tmpPath)
				editUpdate(ctx, chatJID, msgID, "Build failed: "+buildErr.Error(), pct)
				return nil
			}
			editUpdate(ctx, chatJID, msgID, "Build complete", 90)

			// Step 4 — replace binary
			if err := os.Rename(tmpPath, exePath); err != nil {
				msg := fmt.Sprintf("Built successfully but could not replace binary.\nStop the bot and rename manually:\n%s → %s", tmpPath, exePath)
				editUpdate(ctx, chatJID, msgID, msg, 90)
				return nil
			}
			editUpdate(ctx, chatJID, msgID, "Binary replaced", 95)

			// Step 5 — restart
			text := fmt.Sprintf("Ortsax updated!\n%s\nRestarting...", updateBar(100))
			edit := ctx.Client.BuildEdit(chatJID, msgID, &waProto.Message{Conversation: proto.String(text)})
			ctx.Client.SendMessage(context.Background(), chatJID, edit)
			time.Sleep(600 * time.Millisecond)

			if restartFunc != nil {
				go restartFunc()
			}
			return nil
		},
	})
}
