package plugins

import (
	"context"
	"fmt"
	"strconv"

	"go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

func init() {
	Register(&Command{
		Pattern:  "report",
		Category: "utility",
		Func:     reportCmd,
	})
}

func reportCmd(ctx *Context) error {
	ci := ctx.Event.Message.GetExtendedTextMessage().GetContextInfo()
	msgID := ci.GetStanzaID()
	participant := ci.GetParticipant()

	if msgID == "" {
		ctx.Reply(T().ReportUsage)
		return nil
	}

	chat := ctx.Event.Info.Chat
	isGroup := chat.Server == types.GroupServer

	var senderJID types.JID
	if participant != "" {
		if parsed, err := types.ParseJID(participant); err == nil {
			senderJID = parsed.ToNonAD()
		}
	} else if !isGroup {
		senderJID = chat.ToNonAD()
	}

	ts := uint64(ctx.Event.Info.Timestamp.Unix())

	// Build the spam_list node matching WhatsApp's wire format.
	msgAttrs := waBinary.Attrs{
		"id": msgID,
		"t":  strconv.FormatUint(ts, 10),
	}
	if !senderJID.IsEmpty() {
		msgAttrs["from"] = senderJID.String()
	}
	if isGroup && !senderJID.IsEmpty() {
		msgAttrs["participant"] = senderJID.String()
	}

	spamListAttrs := waBinary.Attrs{"spam_flow": "MessageMenu"}
	if isGroup {
		spamListAttrs["jid"] = chat.String()
		if gi, err := ctx.Client.GetGroupInfo(context.Background(), chat); err == nil {
			spamListAttrs["subject"] = gi.Name
		}
	}

	spamListNode := waBinary.Node{
		Tag:   "spam_list",
		Attrs: spamListAttrs,
		Content: []waBinary.Node{
			{Tag: "message", Attrs: msgAttrs},
		},
	}

	// whatsmeow has no stable public API for sending arbitrary IQ stanzas;
	// DangerousInternals is the only available escape hatch.
	//nolint:staticcheck
	internal := ctx.Client.DangerousInternals()
	resp, err := internal.SendIQ(context.Background(), whatsmeow.DangerousInfoQuery{ //nolint:staticcheck
		Namespace: "spam",
		Type:      "set",
		To:        types.ServerJID,
		Content:   []waBinary.Node{spamListNode},
	})
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().ReportFailed, err.Error()))
		return nil
	}

	reportID := ""
	if resp != nil {
		if child, ok := resp.GetOptionalChildByTag("report_id"); ok {
			if s, ok := child.Content.(string); ok {
				reportID = s
			}
		}
	}
	if reportID != "" {
		ctx.Reply(fmt.Sprintf(T().ReportDone, reportID))
	} else {
		ctx.Reply(T().ReportDoneNoID)
	}
	return nil
}
