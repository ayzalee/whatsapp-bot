package plugins

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func init() {
	Register(&Command{
		Pattern:  "whois",
		Category: "utility",
		Func:     whoisCmd,
	})
}

func whoisCmd(ctx *Context) error {
	var targetJID types.JID

	ci := ctx.Event.Message.GetExtendedTextMessage().GetContextInfo()
	if ci.GetParticipant() != "" {
		parsed, err := types.ParseJID(ci.GetParticipant())
		if err == nil {
			targetJID = parsed.ToNonAD()
		}
	} else if len(ci.GetMentionedJID()) > 0 {
		parsed, err := types.ParseJID(ci.GetMentionedJID()[0])
		if err == nil {
			targetJID = parsed.ToNonAD()
		}
	} else if ctx.Text != "" {

		num := strings.TrimPrefix(strings.TrimSpace(ctx.Text), "+")
		parsed, err := types.ParseJID(num + "@s.whatsapp.net")
		if err == nil {
			targetJID = parsed.ToNonAD()
		}
	}

	if targetJID.IsEmpty() {
		targetJID = ctx.Event.Info.Sender.ToNonAD()
	}

	infoMap, err := ctx.Client.GetUserInfo(context.Background(), []types.JID{targetJID})
	if err != nil {
		ctx.Reply(T().WhoisFailed)
		return nil
	}

	info, ok := infoMap[targetJID]
	if !ok {
		ctx.Reply(T().WhoisNotFound)
		return nil
	}

	contact, _ := ctx.Client.Store.Contacts.GetContact(context.Background(), targetJID)
	name := contact.FullName
	if name == "" {
		name = contact.PushName
	}
	if name == "" {
		name = "Unknown"
	}

	status := info.Status
	if status == "" {
		status = "No status"
	}

	caption := fmt.Sprintf(T().WhoisCaption, targetJID.User, name, status, len(info.Devices))

	picInfo, err := ctx.Client.GetProfilePictureInfo(context.Background(), targetJID, &whatsmeow.GetProfilePictureParams{
		Preview: false,
	})

	if err != nil || picInfo == nil {

		ctx.Reply(caption)
		return nil
	}

	resp, err := http.Get(picInfo.URL)
	if err != nil {
		ctx.Reply(caption)
		return nil
	}
	defer resp.Body.Close()
	imgData, err := io.ReadAll(resp.Body)
	if err != nil {
		ctx.Reply(caption)
		return nil
	}

	uploaded, err := ctx.Client.Upload(context.Background(), imgData, whatsmeow.MediaImage)
	if err != nil {
		ctx.Reply(caption)
		return nil
	}

	_ = time.Now()

	id := ctx.Client.GenerateMessageID()
	sendQueue <- sendTask{
		client: ctx.Client,
		to:     ctx.Event.Info.Chat,
		msg: &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(imgData))),
				Mimetype:      proto.String("image/jpeg"),
				Caption:       proto.String(caption),
			},
		},
		id: id,
	}
	return nil
}
