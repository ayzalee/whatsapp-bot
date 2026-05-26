package plugins

func init() {
	Register(&Command{
		Pattern:  "jid",
		Aliases:  []string{"getjid"},
		IsSudo:   true,
		Category: "utility",
		Func: func(ctx *Context) error {
			quoted := quotedMsg(ctx)
			if quoted != nil {

				phone := ctx.Event.Info.SenderAlt.User
				if phone == "" {

					phone = GetAltID(ctx.Event.Info.Sender.String())
				}
				if phone == "" {
					phone = ctx.Event.Info.Sender.User
				}
				ctx.Reply(phone + "@s.whatsapp.net")
			} else {
				ctx.Reply(ctx.Event.Info.Chat.String())
			}
			return nil
		},
	})
}
