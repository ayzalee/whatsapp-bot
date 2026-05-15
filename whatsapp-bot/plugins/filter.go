package plugins

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterModerationHook(filterHook)

	Register(&Command{
		Pattern:  "filter",
		Category: "utility",
		Func:     filterListCmd,
	})

	Register(&Command{
		Pattern:  "gfilter",
		Category: "utility",
		Func:     gfilterCmd,
	})

	Register(&Command{
		Pattern:  "dfilter",
		Category: "utility",
		Func:     dfilterCmd,
	})
}

// filterHook auto-replies when an incoming message matches a registered filter.
func filterHook(client *whatsmeow.Client, evt *events.Message) {
	text := extractMsgText(evt)
	if text == "" {
		return
	}
	isGroup := evt.Info.Chat.Server == types.GroupServer
	chatJID := evt.Info.Chat.String()

	var response string
	var found bool

	if isGroup {
		response, found = matchFilter("group", chatJID, text)
	} else {
		response, found = matchFilter("dm", "dm", text)
	}

	if found {
		client.SendMessage(context.Background(), evt.Info.Chat,
			&waProto.Message{Conversation: proto.String(response)})
	}
}

// filterListCmd shows all active filters for the current chat.
func filterListCmd(ctx *Context) error {
	isGroup := ctx.Event.Info.Chat.Server == types.GroupServer
	var filters map[string]string
	if isGroup {
		filters = getFilters("group", ctx.Event.Info.Chat.String())
	} else {
		filters = getFilters("dm", "dm")
	}
	if len(filters) == 0 {
		ctx.Reply(T().FilterNone)
		return nil
	}
	var sb strings.Builder
	for k, v := range filters {
		sb.WriteString(fmt.Sprintf("*%s* → %s\n", k, v))
	}
	ctx.Reply(fmt.Sprintf(T().FilterList, strings.TrimRight(sb.String(), "\n")))
	return nil
}

// gfilterCmd handles .gfilter set/get/del for group-scoped filters.
func gfilterCmd(ctx *Context) error {
	return handleFilterCmd(ctx, "group", ctx.Event.Info.Chat.String())
}

// dfilterCmd handles .dfilter set/get/del for DM-scoped filters.
func dfilterCmd(ctx *Context) error {
	return handleFilterCmd(ctx, "dm", "dm")
}

func handleFilterCmd(ctx *Context, scope, chatJID string) error {
	if len(ctx.Args) == 0 {
		filters := getFilters(scope, chatJID)
		if len(filters) == 0 {
			ctx.Reply(T().FilterNone)
			return nil
		}
		var sb strings.Builder
		for k, v := range filters {
			sb.WriteString(fmt.Sprintf("*%s* → %s\n", k, v))
		}
		ctx.Reply(fmt.Sprintf(T().FilterList, strings.TrimRight(sb.String(), "\n")))
		return nil
	}

	sub := strings.ToLower(ctx.Args[0])
	rest := strings.TrimSpace(strings.TrimPrefix(ctx.Text, ctx.Args[0]))

	switch sub {
	case "set":
		idx := strings.IndexByte(rest, '|')
		if idx < 0 || strings.TrimSpace(rest[:idx]) == "" {
			ctx.Reply(T().FilterSetUsage)
			return nil
		}
		keyword := strings.TrimSpace(rest[:idx])
		response := strings.TrimSpace(rest[idx+1:])
		if response == "" {
			ctx.Reply(T().FilterSetUsage)
			return nil
		}
		setFilter(scope, chatJID, keyword, response)
		ctx.Reply(fmt.Sprintf(T().FilterSet, keyword))

	case "del":
		keyword := strings.TrimSpace(rest)
		if keyword == "" {
			ctx.Reply(T().FilterDelUsage)
			return nil
		}
		if delFilter(scope, chatJID, keyword) {
			ctx.Reply(fmt.Sprintf(T().FilterDeleted, keyword))
		} else {
			ctx.Reply(fmt.Sprintf(T().FilterNotFound, keyword))
		}

	case "get":
		keyword := strings.TrimSpace(rest)
		filters := getFilters(scope, chatJID)
		if resp, ok := filters[keyword]; ok {
			ctx.Reply(fmt.Sprintf("*%s* → %s", keyword, resp))
		} else {
			ctx.Reply(fmt.Sprintf(T().FilterNotFound, keyword))
		}

	default:
		ctx.Reply(T().FilterSetUsage)
	}
	return nil
}
