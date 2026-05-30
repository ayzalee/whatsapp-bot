package plugins

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func init() {
	Register(&Command{
		Pattern:  "dl",
		Category: "download",
		Func:     dlCmd,
	})
}

func dlCmd(ctx *Context) error {
	input := strings.TrimSpace(ctx.Text)
	if input == "" {
		ctx.Reply(T().DlUsage)
		return nil
	}

	tmpDir, err := os.MkdirTemp("", "dl-*")
	if err != nil {
		ctx.Reply("Failed to create temp directory.")
		return nil
	}
	defer os.RemoveAll(tmpDir)

	outTemplate := filepath.Join(tmpDir, "%(title).50s.%(ext)s")

	var args []string
	isAudio := false
	isURL := strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://")

	baseFlags := []string{
		"--no-playlist",
		"--no-warnings", "--quiet",
		"--concurrent-fragments", "4",
		"--no-part",
		"--remote-components", "ejs:github",
		"--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
	}

	if strings.HasPrefix(input, "mp3 ") {
		isAudio = true
		url := strings.TrimPrefix(input, "mp3 ")
		args = append([]string{
			"-x", "--audio-format", "mp3", "--audio-quality", "0",
		}, baseFlags...)
		args = append(args, "-o", outTemplate, url)

	} else if isURL {
		args = append([]string{
			"-f", "best[ext=mp4]/best",
			"--merge-output-format", "mp4",
		}, baseFlags...)
		args = append(args, "-o", outTemplate, input)

	} else {
		isAudio = true
		args = append([]string{
			"-f", "bestaudio/best",
			"-x", "--audio-format", "mp3", "--audio-quality", "0",
			"--default-search", "ytsearch",
		}, baseFlags...)
		args = append(args, "-o", outTemplate, input)
	}

	if dlCookieFile != "" {
		args = append(args, "--cookies", dlCookieFile)
	}
	cookieFile := dlCookieFile
	if cookieFile == "" {
		if _, err := os.Stat("cookies.txt"); err == nil {
			cookieFile = "cookies.txt"
		}
	}
	if cookieFile != "" {
		args = append(args, "--cookies", cookieFile)
	}
	cmd := exec.Command("yt-dlp", args...)
	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		ctx.Reply("Error (cookie=" + cookieFile + "): " + string(out))
		return nil
	}

	files, err := filepath.Glob(filepath.Join(tmpDir, "*"))
	if err != nil || len(files) == 0 {
		ctx.Reply(T().DlNoFile)
		return nil
	}

	filePath := files[0]
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		ctx.Reply("Failed to read file.")
		return nil
	}

	msgID := ctx.Event.Info.ID
	senderJID := ctx.Event.Info.Sender.String()
	chatJID := ctx.Event.Info.Chat.String()

	contextInfo := &waProto.ContextInfo{
		StanzaID:      proto.String(msgID),
		Participant:   proto.String(senderJID),
		QuotedMessage: &waProto.Message{Conversation: proto.String("")},
		RemoteJID:     proto.String(chatJID),
	}

	id := ctx.Client.GenerateMessageID()

	if isAudio {
		uploaded, err := ctx.Client.Upload(context.Background(), fileBytes, whatsmeow.MediaAudio)
		if err != nil {
			ctx.Reply("Failed to upload audio.")
			return nil
		}
		sendQueue <- sendTask{
			client: ctx.Client,
			to:     ctx.Event.Info.Chat,
			msg: &waProto.Message{
				AudioMessage: &waProto.AudioMessage{
					URL:           proto.String(uploaded.URL),
					DirectPath:    proto.String(uploaded.DirectPath),
					MediaKey:      uploaded.MediaKey,
					FileEncSHA256: uploaded.FileEncSHA256,
					FileSHA256:    uploaded.FileSHA256,
					FileLength:    proto.Uint64(uint64(len(fileBytes))),
					Mimetype:      proto.String("audio/mpeg"),
					ContextInfo:   contextInfo,
				},
			},
			id: id,
		}
	} else {
		uploaded, err := ctx.Client.Upload(context.Background(), fileBytes, whatsmeow.MediaVideo)
		if err != nil {
			ctx.Reply("Failed to upload video.")
			return nil
		}
		sendQueue <- sendTask{
			client: ctx.Client,
			to:     ctx.Event.Info.Chat,
			msg: &waProto.Message{
				VideoMessage: &waProto.VideoMessage{
					URL:           proto.String(uploaded.URL),
					DirectPath:    proto.String(uploaded.DirectPath),
					MediaKey:      uploaded.MediaKey,
					FileEncSHA256: uploaded.FileEncSHA256,
					FileSHA256:    uploaded.FileSHA256,
					FileLength:    proto.Uint64(uint64(len(fileBytes))),
					Mimetype:      proto.String("video/mp4"),
					ContextInfo:   contextInfo,
				},
			},
			id: id,
		}
	}

	return nil
}
