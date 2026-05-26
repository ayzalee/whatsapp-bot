package plugins

import (
	"context"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type Command struct {
	Pattern  string
	Aliases  []string
	IsSudo   bool
	IsAdmin  bool
	IsGroup  bool
	Category string
	Func     func(ctx *Context) error
}

type Context struct {
	Client     *whatsmeow.Client
	Event      *events.Message
	Args       []string  
	Text       string    
	Prefix     string    
	Matched    string    
	ReceivedAt time.Time 
}

func (c *Context) Reply(text string) (whatsmeow.SendResponse, error) {
	id := c.Client.GenerateMessageID()
	sendQueue <- sendTask{
		client: c.Client,
		to:     c.Event.Info.Chat,
		msg:    &waProto.Message{Conversation: proto.String(text)},
		id:     id,
	}
	return whatsmeow.SendResponse{ID: id}, nil
}

func (c *Context) ReplySync(text string) (whatsmeow.SendResponse, error) {
	return c.Client.SendMessage(context.Background(), c.Event.Info.Chat,
		&waProto.Message{Conversation: proto.String(text)},
		whatsmeow.SendRequestExtra{Timeout: sendTimeout},
	)
}

func (c *Context) QueueEdit(originalID types.MessageID, newText string) {
	sendQueue <- sendTask{
		client: c.Client,
		to:     c.Event.Info.Chat,
		msg: c.Client.BuildEdit(c.Event.Info.Chat, originalID, &waProto.Message{
			Conversation: proto.String(newText),
		}),
		id: c.Client.GenerateMessageID(),
	}
}

var registry []*Command

var registryMap = make(map[string]*Command)

var categoryMap = make(map[string][]*Command)

func Register(cmd *Command) {
	registry = append(registry, cmd)
	registryMap[strings.ToLower(cmd.Pattern)] = cmd
	for _, alias := range cmd.Aliases {
		registryMap[strings.ToLower(alias)] = cmd
	}
	cat := strings.ToLower(cmd.Category)
	if cat == "" {
		cat = "general"
	}
	categoryMap[cat] = append(categoryMap[cat], cmd)
}

func parseCommand(text string, prefixes []string) (prefix, name, rest string, ok bool) {
	lower := strings.ToLower(text)
	for _, p := range prefixes {
		var afterOrig, afterLower string
		if p == "" {
			afterOrig = text
			afterLower = lower
		} else {
			lp := strings.ToLower(p)
			if !strings.HasPrefix(lower, lp) {
				continue
			}
			afterOrig = text[len(lp):]
			afterLower = lower[len(lp):]
		}
		afterLower = strings.TrimLeft(afterLower, " ")
		if afterLower == "" {
			continue
		}
		
		trimmed := len(afterOrig) - len(strings.TrimLeft(afterOrig, " "))
		afterOrig = afterOrig[trimmed:]
		if i := strings.IndexByte(afterLower, ' '); i != -1 {
			name = afterLower[:i]
			rest = strings.TrimSpace(afterOrig[i+1:])
		} else {
			name = afterLower
		}
		return p, name, rest, true
	}
	return "", "", "", false
}

func findCommand(name string) *Command {
	return registryMap[name]
}

func extractText(evt *events.Message) string {
	if t := evt.Message.GetConversation(); t != "" {
		return t
	}
	if t := evt.Message.GetExtendedTextMessage().GetText(); t != "" {
		return t
	}
	if t := evt.Message.GetImageMessage().GetCaption(); t != "" {
		return t
	}
	if t := evt.Message.GetVideoMessage().GetCaption(); t != "" {
		return t
	}
	return ""
}

func Dispatch(client *whatsmeow.Client, evt *events.Message) {
	receivedAt := time.Now() 
	text := extractText(evt)
	if text == "" {
		return
	}

	senderID := evt.Info.Sender.User 
	isGroup := evt.Info.Chat.Server == types.GroupServer

	
	if isGroup && BotSettings.IsGCDisabled() {
		return
	}

	prefix, name, rest, ok := parseCommand(text, BotSettings.GetPrefixes())
	if !ok {
		return
	}

	cmd := findCommand(name)
	if cmd == nil {
		
		if menu := CategoryMenu(name); menu != "" {
			miniCtx := &Context{Client: client, Event: evt}
			miniCtx.Reply(menu)
		}
		return
	}

	ctx := &Context{
		Client:     client,
		Event:      evt,
		Args:       strings.Fields(rest),
		Text:       rest,
		Prefix:     prefix,
		Matched:    name,
		ReceivedAt: receivedAt,
	}

	isSudo := BotSettings.IsSudo(senderID)
	
	if !isSudo && evt.Info.SenderAlt.User != "" {
		isSudo = BotSettings.IsSudo(evt.Info.SenderAlt.User)
	}

	
	isBanned := BotSettings.IsBanned(senderID)
	if !isBanned && evt.Info.SenderAlt.User != "" {
		isBanned = BotSettings.IsBanned(evt.Info.SenderAlt.User)
	}
	if isBanned {
		return
	}
	mode := BotSettings.GetMode()

	
	if mode == ModePrivate && !isSudo {
		return
	}

	if cmd.IsGroup && !isGroup {
		ctx.Reply(T().GroupOnly)
		return
	}

	if cmd.IsSudo && !isSudo {
		ctx.Reply(T().SudoOnly)
		return
	}

	if cmd.IsAdmin && isGroup {
		botJID := client.Store.ID.ToNonAD()
		group, err := client.GetGroupInfo(context.Background(), evt.Info.Chat)
		if err == nil {
			if !botIsAdmin(group.Participants, ownerPhone, botJID.User) {
				ctx.Reply(T().BotNotAdmin)
				return
			}
			if !isSudo {
				p := findParticipant(group.Participants, evt.Info.Sender.User, evt.Info.SenderAlt.User)
				if p == nil || (!p.IsAdmin && !p.IsSuperAdmin) {
					ctx.Reply(T().SenderNotAdmin)
					return
				}
			}
		}
	}

	if BotSettings.IsCmdDisabled(name) {
		ctx.Reply(T().CmdIsDisabled)
		return
	}

	_ = cmd.Func(ctx)
}
