package plugins

import (
	"fmt"
	"strings"
)

func init() {
	Register(&Command{
		Pattern:  "lang",
		Category: "settings",
		Func: func(ctx *Context) error {
			// No argument – show current language and full list (available to anyone).
			if ctx.Text == "" {
				name := LangNames[BotSettings.GetLanguage()]
				ctx.Reply(fmt.Sprintf(T().LangCurrent, name) + "\n\n" + langList())
				return nil
			}

			// Changing language is restricted to sudo users.
			if !BotSettings.IsSudo(ctx.Event.Info.Sender.User) {
				ctx.Reply(T().SudoOnly)
				return nil
			}

			code := strings.ToLower(strings.TrimSpace(ctx.Text))
			if _, ok := translations[code]; !ok {
				ctx.Reply(fmt.Sprintf(T().LangUnknown, code, availableLangs()))
				return nil
			}

			BotSettings.SetLanguage(code)
			if err := SaveSettings(); err != nil {
				ctx.Reply(fmt.Sprintf(T().SaveFailed, err.Error()))
				return err
			}

			ctx.Reply(fmt.Sprintf(T().LangSet, LangNames[code]))
			return nil
		},
	})
}
