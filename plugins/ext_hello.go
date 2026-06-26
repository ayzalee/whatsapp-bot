package plugins

func init() {
	Register(&Command{
		Pattern:  "hello",
		Aliases:  []string{"hi"},
		Category: "general",
		Func: func(ctx *Context) error {
			name := ctx.Event.Info.PushName
			if name == "" {
				name = ctx.Event.Info.Sender.User
			}
			ctx.Reply("👋 Hello, " + name + "! External plugin works ✅")
			return nil
		},
	})
}