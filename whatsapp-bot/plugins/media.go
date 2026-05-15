package plugins

import (
	"context"
	"fmt"
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
		Pattern:  "mp3",
		Category: "media",
		Func:     mp3Cmd,
	})
	Register(&Command{
		Pattern:  "black",
		Category: "media",
		Func:     blackCmd,
	})
	Register(&Command{
		Pattern:  "trim",
		Category: "media",
		Func:     trimCmd,
	})
}

// quotedMsg returns the quoted message from the event's ContextInfo,
// or nil if the user did not reply to a message.
func quotedMsg(ctx *Context) *waProto.Message {
	return ctx.Event.Message.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
}

// runFFmpeg runs ffmpeg with the given arguments and returns stdout+stderr on failure.
func runFFmpeg(args ...string) error {
	cmd := exec.Command("ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w\n%s", err, string(out))
	}
	return nil
}

// mp3Cmd converts quoted audio/video to an mp3 file and sends it.
func mp3Cmd(ctx *Context) error {
	quoted := quotedMsg(ctx)
	if quoted == nil {
		ctx.Reply(T().MediaNoReply)
		return nil
	}

	// accept audio or video
	var data []byte
	var err error
	if quoted.GetAudioMessage() != nil {
		data, err = ctx.Client.Download(context.Background(), quoted.GetAudioMessage())
	} else if quoted.GetVideoMessage() != nil {
		data, err = ctx.Client.Download(context.Background(), quoted.GetVideoMessage())
	} else {
		ctx.Reply(T().MediaNoReply)
		return nil
	}
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	ctx.Reply(T().MediaProcessing)

	tmp, err := os.MkdirTemp("", "zaelix-mp3-*")
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}
	defer os.RemoveAll(tmp)

	inFile := filepath.Join(tmp, "input")
	outFile := filepath.Join(tmp, "output.mp3")

	if err = os.WriteFile(inFile, data, 0600); err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	if err = runFFmpeg("-y", "-i", inFile, "-vn", "-ar", "44100", "-ac", "2", "-b:a", "192k", outFile); err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	mp3Bytes, err := os.ReadFile(outFile)
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	resp, err := ctx.Client.Upload(context.Background(), mp3Bytes, whatsmeow.MediaAudio)
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	audioMsg := &waProto.AudioMessage{
		Mimetype:      proto.String("audio/mpeg"),
		URL:           proto.String(resp.URL),
		DirectPath:    proto.String(resp.DirectPath),
		MediaKey:      resp.MediaKey,
		FileEncSHA256: resp.FileEncSHA256,
		FileSHA256:    resp.FileSHA256,
		FileLength:    proto.Uint64(resp.FileLength),
	}
	id := ctx.Client.GenerateMessageID()
	sendQueue <- sendTask{
		client: ctx.Client,
		to:     ctx.Event.Info.Chat,
		msg:    &waProto.Message{AudioMessage: audioMsg},
		id:     id,
	}
	return nil
}

// blackCmd converts quoted audio to a black-screen video.
func blackCmd(ctx *Context) error {
	quoted := quotedMsg(ctx)
	if quoted == nil || quoted.GetAudioMessage() == nil {
		ctx.Reply(T().MediaNoReply)
		return nil
	}

	data, err := ctx.Client.Download(context.Background(), quoted.GetAudioMessage())
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	ctx.Reply(T().MediaProcessing)

	tmp, err := os.MkdirTemp("", "zaelix-black-*")
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}
	defer os.RemoveAll(tmp)

	inFile := filepath.Join(tmp, "input")
	outFile := filepath.Join(tmp, "output.mp4")

	if err = os.WriteFile(inFile, data, 0600); err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	if err = runFFmpeg(
		"-y",
		"-f", "lavfi", "-i", "color=c=black:s=640x360:r=25",
		"-i", inFile,
		"-shortest",
		"-c:v", "libx264", "-c:a", "aac",
		outFile,
	); err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	mp4Bytes, err := os.ReadFile(outFile)
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	resp, err := ctx.Client.Upload(context.Background(), mp4Bytes, whatsmeow.MediaVideo)
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	videoMsg := &waProto.VideoMessage{
		Mimetype:      proto.String("video/mp4"),
		URL:           proto.String(resp.URL),
		DirectPath:    proto.String(resp.DirectPath),
		MediaKey:      resp.MediaKey,
		FileEncSHA256: resp.FileEncSHA256,
		FileSHA256:    resp.FileSHA256,
		FileLength:    proto.Uint64(resp.FileLength),
	}
	id := ctx.Client.GenerateMessageID()
	sendQueue <- sendTask{
		client: ctx.Client,
		to:     ctx.Event.Info.Chat,
		msg:    &waProto.Message{VideoMessage: videoMsg},
		id:     id,
	}
	return nil
}

// trimCmd trims quoted audio or video. Args: <start> [end]
// Times can be in seconds (e.g., "10") or mm:ss (e.g., "1:30").
func trimCmd(ctx *Context) error {
	if len(ctx.Args) < 1 {
		ctx.Reply(T().TrimUsage)
		return nil
	}

	start := ctx.Args[0]
	end := ""
	if len(ctx.Args) >= 2 {
		end = ctx.Args[1]
	}

	quoted := quotedMsg(ctx)
	if quoted == nil {
		ctx.Reply(T().MediaNoReply)
		return nil
	}

	isAudio := quoted.GetAudioMessage() != nil
	isVideo := quoted.GetVideoMessage() != nil
	if !isAudio && !isVideo {
		ctx.Reply(T().MediaNoReply)
		return nil
	}

	var data []byte
	var err error
	if isAudio {
		data, err = ctx.Client.Download(context.Background(), quoted.GetAudioMessage())
	} else {
		data, err = ctx.Client.Download(context.Background(), quoted.GetVideoMessage())
	}
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	ctx.Reply(T().MediaProcessing)

	ext := ".mp3"
	mediaType := whatsmeow.MediaAudio
	if isVideo {
		ext = ".mp4"
		mediaType = whatsmeow.MediaVideo
	}

	tmp, err := os.MkdirTemp("", "zaelix-trim-*")
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}
	defer os.RemoveAll(tmp)

	inFile := filepath.Join(tmp, "input")
	outFile := filepath.Join(tmp, "output"+ext)

	if err = os.WriteFile(inFile, data, 0600); err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	ffArgs := []string{"-y", "-ss", start}
	if end != "" {
		ffArgs = append(ffArgs, "-to", end)
	}
	ffArgs = append(ffArgs, "-i", inFile, "-c", "copy", outFile)

	if err = runFFmpeg(ffArgs...); err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	outBytes, err := os.ReadFile(outFile)
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	uploadResp, err := ctx.Client.Upload(context.Background(), outBytes, mediaType)
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	id := ctx.Client.GenerateMessageID()
	var msg *waProto.Message

	if isAudio {
		mime := quoted.GetAudioMessage().GetMimetype()
		if mime == "" {
			mime = "audio/mpeg"
		}
		// keep ogg/opus as-is if the source was ogg (trim -c copy preserves codec)
		if strings.HasSuffix(ext, ".mp3") && strings.Contains(mime, "ogg") {
			outFile = strings.TrimSuffix(outFile, ".mp3") + ".ogg"
			mime = "audio/ogg; codecs=opus"
		}
		msg = &waProto.Message{
			AudioMessage: &waProto.AudioMessage{
				Mimetype:      proto.String(mime),
				URL:           proto.String(uploadResp.URL),
				DirectPath:    proto.String(uploadResp.DirectPath),
				MediaKey:      uploadResp.MediaKey,
				FileEncSHA256: uploadResp.FileEncSHA256,
				FileSHA256:    uploadResp.FileSHA256,
				FileLength:    proto.Uint64(uploadResp.FileLength),
			},
		}
	} else {
		mime := quoted.GetVideoMessage().GetMimetype()
		if mime == "" {
			mime = "video/mp4"
		}
		msg = &waProto.Message{
			VideoMessage: &waProto.VideoMessage{
				Mimetype:      proto.String(mime),
				URL:           proto.String(uploadResp.URL),
				DirectPath:    proto.String(uploadResp.DirectPath),
				MediaKey:      uploadResp.MediaKey,
				FileEncSHA256: uploadResp.FileEncSHA256,
				FileSHA256:    uploadResp.FileSHA256,
				FileLength:    proto.Uint64(uploadResp.FileLength),
			},
		}
	}

	sendQueue <- sendTask{
		client: ctx.Client,
		to:     ctx.Event.Info.Chat,
		msg:    msg,
		id:     id,
	}
	return nil
}
