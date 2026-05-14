package plugins

import (
	"context"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func init() {
	Register(&Command{
		Pattern:  "block",
		Category: "utility",
		Func: func(ctx *Context) error {
			arg := ctx.Text
			if arg == "" {
				ctx.Reply(T().BlockUsage)
				return nil
			}
			phone, _ := ResolveTarget(ctx, arg)
			if phone == "" {
				ctx.Reply(T().UserResolveFail)
				return nil
			}
			jid := types.NewJID(phone, types.DefaultUserServer)
			ctx.Client.UpdateBlocklist(context.Background(), jid, events.BlocklistChangeActionBlock)
			ctx.Reply(T().BlockOK)
			return nil
		},
	})

	Register(&Command{
		Pattern:  "unblock",
		Category: "utility",
		Func: func(ctx *Context) error {
			arg := ctx.Text
			if arg == "" {
				ctx.Reply(T().UnblockUsage)
				return nil
			}
			phone, _ := ResolveTarget(ctx, arg)
			if phone == "" {
				ctx.Reply(T().UserResolveFail)
				return nil
			}
			jid := types.NewJID(phone, types.DefaultUserServer)
			ctx.Client.UpdateBlocklist(context.Background(), jid, events.BlocklistChangeActionUnblock)
			ctx.Reply(T().UnblockOK)
			return nil
		},
	})
}
