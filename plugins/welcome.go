package plugins

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type greetSettings struct {
	enabled bool
	message string
}

var welcomeChats = map[string]*greetSettings{}
var goodbyeChats = map[string]*greetSettings{}

const defaultWelcomeMsg = "&mention welcome to &name 🎉"
const defaultGoodbyeMsg = "&mention left &name 👋"

func init() {
	Register(&Command{
		Pattern:  "welcome",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			return handleGreetCmd(ctx, welcomeChats, defaultWelcomeMsg, "welcome")
		},
	})

	Register(&Command{
		Pattern:  "goodbye",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			return handleGreetCmd(ctx, goodbyeChats, defaultGoodbyeMsg, "goodbye")
		},
	})
}

func handleGreetCmd(ctx *Context, store map[string]*greetSettings, defaultMsg, cmdName string) error {
	chatJID := ctx.Event.Info.Chat.String()
	arg := strings.TrimSpace(ctx.Text)
	lower := strings.ToLower(arg)

	switch {
	case lower == "on":
		s := store[chatJID]
		if s == nil {
			s = &greetSettings{message: defaultMsg}
			store[chatJID] = s
		}
		s.enabled = true
		ctx.Reply(fmt.Sprintf(T().GreetEnabled, strings.Title(cmdName)))

	case lower == "off":
		if s := store[chatJID]; s != nil {
			s.enabled = false
		}
		ctx.Reply(fmt.Sprintf(T().GreetDisabled, strings.Title(cmdName)))

	case arg == "":
		s := store[chatJID]
		status := "off"
		msg := defaultMsg
		if s != nil {
			if s.enabled {
				status = "on"
			}
			msg = s.message
		}
		ctx.Reply(fmt.Sprintf(T().GreetStatus, strings.Title(cmdName), status, msg, cmdName, cmdName, cmdName))

	case strings.HasPrefix(lower, "set "):
		newMsg := strings.TrimSpace(arg[4:])
		if newMsg == "" {
			ctx.Reply(fmt.Sprintf(T().GreetSetUsage, cmdName))
			return nil
		}
		s := store[chatJID]
		if s == nil {
			s = &greetSettings{}
			store[chatJID] = s
		}
		s.enabled = true
		s.message = newMsg
		ctx.Reply(fmt.Sprintf(T().GreetSetOK, strings.Title(cmdName)))

	default:
		ctx.Reply(fmt.Sprintf(T().GreetUsage, cmdName))
	}
	return nil
}

func renderGreetTemplate(client *whatsmeow.Client, chat types.JID, user types.JID, template string) (text string, mentions []string, withPic bool) {
	text = template
	mentions = []string{}

	if strings.Contains(text, "&mention") {
		text = strings.ReplaceAll(text, "&mention", "@"+user.User)
		mentions = append(mentions, user.ToNonAD().String())
	}

	if strings.Contains(text, "&name") || strings.Contains(text, "&desc") || strings.Contains(text, "&size") {
		info, err := client.GetGroupInfo(context.Background(), chat)
		if err == nil {
			text = strings.ReplaceAll(text, "&name", info.Name)
			text = strings.ReplaceAll(text, "&desc", info.Topic)
			text = strings.ReplaceAll(text, "&size", strconv.Itoa(len(info.Participants)))
		}
	}

	if strings.Contains(text, "&pp") {
		text = strings.ReplaceAll(text, "&pp", "")
		withPic = true
	}

	text = strings.TrimSpace(text)
	return text, mentions, withPic
}

func sendGreetMessage(client *whatsmeow.Client, chat types.JID, user types.JID, template string) {
	text, mentions, withPic := renderGreetTemplate(client, chat, user, template)
	if text == "" {
		return
	}

	if !withPic {
		id := client.GenerateMessageID()
		sendQueue <- sendTask{
			client: client,
			to:     chat,
			msg: &waProto.Message{
				ExtendedTextMessage: &waProto.ExtendedTextMessage{
					Text: proto.String(text),
					ContextInfo: &waProto.ContextInfo{
						MentionedJID: mentions,
					},
				},
			},
			id: client.GenerateMessageID(),
		}
		_ = id
		return
	}

	picInfo, err := client.GetProfilePictureInfo(context.Background(), chat, &whatsmeow.GetProfilePictureParams{Preview: false})
	if err != nil || picInfo == nil {
		id := client.GenerateMessageID()
		sendQueue <- sendTask{
			client: client,
			to:     chat,
			msg: &waProto.Message{
				ExtendedTextMessage: &waProto.ExtendedTextMessage{
					Text: proto.String(text),
					ContextInfo: &waProto.ContextInfo{
						MentionedJID: mentions,
					},
				},
			},
			id: id,
		}
		return
	}

	resp, err := httpGetBytes(picInfo.URL)
	if err != nil {
		return
	}

	uploaded, err := client.Upload(context.Background(), resp, whatsmeow.MediaImage)
	if err != nil {
		return
	}

	id := client.GenerateMessageID()
	sendQueue <- sendTask{
		client: client,
		to:     chat,
		msg: &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(resp))),
				Mimetype:      proto.String("image/jpeg"),
				Caption:       proto.String(text),
				ContextInfo: &waProto.ContextInfo{
					MentionedJID: mentions,
				},
			},
		},
		id: id,
	}
}

func HandleWelcomeGoodbye(client *whatsmeow.Client, evt *events.GroupInfo) {
	chatJID := evt.JID.String()

	if s := welcomeChats[chatJID]; s != nil && s.enabled {
		for _, jid := range evt.Join {
			sendGreetMessage(client, evt.JID, jid, s.message)
		}
	}

	if s := goodbyeChats[chatJID]; s != nil && s.enabled {
		for _, jid := range evt.Leave {
			sendGreetMessage(client, evt.JID, jid, s.message)
		}
	}
}

func httpGetBytes(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
