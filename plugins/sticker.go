package plugins

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func init() {
	Register(&Command{
		Pattern:  "sticker",
		Aliases:  []string{"s"},
		Category: "media",
		Func:     stickerCmd,
	})
}

func stickerCmd(ctx *Context) error {
	quoted := quotedMsg(ctx)
	if quoted == nil {
		ctx.Reply(T().StickerNoReply)
		return nil
	}

	var imgData []byte
	var err error
	isAnimated := false

	if quoted.GetImageMessage() != nil {
		imgData, err = ctx.Client.Download(context.Background(), quoted.GetImageMessage())
		if quoted.GetImageMessage().GetMimetype() == "image/gif" {
			isAnimated = true
		}
	} else if quoted.GetVideoMessage() != nil {
		imgData, err = ctx.Client.Download(context.Background(), quoted.GetVideoMessage())
		isAnimated = true
	} else if quoted.GetStickerMessage() != nil {
		imgData, err = ctx.Client.Download(context.Background(), quoted.GetStickerMessage())
	} else {
		ctx.Reply(T().StickerNoReply)
		return nil
	}

	if err != nil {
		ctx.Reply(T().StickerNoReply)
		return nil
	}

	tmpDir, _ := os.MkdirTemp("", "sticker-*")
	defer os.RemoveAll(tmpDir)

	inFile := filepath.Join(tmpDir, "input")
	outFile := filepath.Join(tmpDir, "output.webp")

	if err := os.WriteFile(inFile, imgData, 0644); err != nil {
		ctx.Reply(T().StickerNoReply)
		return nil
	}

	var cmd *exec.Cmd
	if isAnimated {

		cmd = exec.Command("ffmpeg", "-i", inFile,
			"-vf", "scale=512:512:force_original_aspect_ratio=decrease,fps=15",
			"-vcodec", "libwebp",
			"-lossless", "0",
			"-compression_level", "6",
			"-quality", "80",
			"-loop", "0",
			"-preset", "default",
			"-an", "-vsync", "0",
			outFile,
		)
	} else {

		cmd = exec.Command("convert", inFile, "-resize", "512x512", "-quality", "80", outFile)
	}

	if err := cmd.Run(); err != nil {
		ctx.Reply(T().StickerNoReply)
		return nil
	}

	webpData, err := os.ReadFile(outFile)
	if err != nil {
		ctx.Reply(T().StickerNoReply)
		return nil
	}

	uploaded, err := ctx.Client.Upload(context.Background(), webpData, whatsmeow.MediaImage)
	if err != nil {
		ctx.Reply(T().StickerNoReply)
		return nil
	}

	msg := &waProto.Message{
		StickerMessage: &waProto.StickerMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(webpData))),
			Mimetype:      proto.String("image/webp"),
			IsAnimated:    proto.Bool(isAnimated),
		},
	}

	id := ctx.Client.GenerateMessageID()
	sendQueue <- sendTask{
		client: ctx.Client,
		to:     ctx.Event.Info.Chat,
		msg:    msg,
		id:     id,
	}
	return nil
}
