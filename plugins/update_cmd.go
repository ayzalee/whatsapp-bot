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

func InitSourceDir(dir string) { botSourceDir = dir }

func SetRestartFunc(f func()) { restartFunc = f }

const updateBarWidth = 22

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

func editUpdate(ctx *Context, chatJID types.JID, msgID, label string, pct int) {
	text := fmt.Sprintf("*Updating Zaelix...* ⏳\n%s\n> %s", updateBar(pct), label)
	edit := ctx.Client.BuildEdit(chatJID, msgID, &waProto.Message{
		Conversation: proto.String(text),
	})
	ctx.Client.SendMessage(context.Background(), chatJID, edit)

	time.Sleep(250 * time.Millisecond)
}

func gitRun(args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", botSourceDir}, args...)...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

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
				ctx.Reply(T().UpdateNoSource)
				return nil
			}

			mode := strings.ToLower(strings.TrimSpace(ctx.Text))

			if mode == "" {
				resp, err := ctx.ReplySync(T().UpdateChecking)
				if err != nil {
					return err
				}
				n, err := pendingCommits()
				if err != nil {
					edit := ctx.Client.BuildEdit(ctx.Event.Info.Chat, resp.ID, &waProto.Message{
						Conversation: proto.String(fmt.Sprintf(T().UpdateCheckFail, err.Error())),
					})
					ctx.Client.SendMessage(context.Background(), ctx.Event.Info.Chat, edit)
					return nil
				}
				var msg string
				if n == 0 {
					msg = T().UpdateUpToDate
				} else {
					msg = fmt.Sprintf(T().UpdateAvailable, n)
				}
				edit := ctx.Client.BuildEdit(ctx.Event.Info.Chat, resp.ID, &waProto.Message{
					Conversation: proto.String(msg),
				})
				ctx.Client.SendMessage(context.Background(), ctx.Event.Info.Chat, edit)
				return nil
			}

			if mode != "now" {
				ctx.Reply(T().UpdateUsage)
				return nil
			}

			chatJID := ctx.Event.Info.Chat
			resp, err := ctx.ReplySync(T().UpdateStarting + "\n" + updateBar(0) + "\n> Starting...")
			if err != nil {
				return err
			}
			msgID := resp.ID

			editUpdate(ctx, chatJID, msgID, "Fetching latest changes...", 5)
			if err := gitRun("fetch", "origin", "--quiet"); err != nil {
				editUpdate(ctx, chatJID, msgID, "Failed to fetch: "+err.Error(), 5)
				return nil
			}
			editUpdate(ctx, chatJID, msgID, "Fetch complete", 15)

			out, _ := exec.Command("git", "-C", botSourceDir, "rev-list", "HEAD..FETCH_HEAD", "--count").Output()
			if strings.TrimSpace(string(out)) == "0" {
				text := T().UpdateAlready + "\n" + updateBar(100)
				edit := ctx.Client.BuildEdit(chatJID, msgID, &waProto.Message{Conversation: proto.String(text)})
				ctx.Client.SendMessage(context.Background(), chatJID, edit)
				return nil
			}

			editUpdate(ctx, chatJID, msgID, "Pulling changes...", 20)
			if err := gitRun("pull", "--ff-only"); err != nil {
				editUpdate(ctx, chatJID, msgID, "Pull failed: "+err.Error(), 20)
				return nil
			}
			editUpdate(ctx, chatJID, msgID, "Changes pulled", 45)

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

			if err := os.Rename(tmpPath, exePath); err != nil {
				msg := fmt.Sprintf(T().UpdateBinaryFail, tmpPath, exePath)
				editUpdate(ctx, chatJID, msgID, msg, 90)
				return nil
			}
			editUpdate(ctx, chatJID, msgID, "Binary replaced", 95)

			text := T().UpdateDone + "\n" + updateBar(100)
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
