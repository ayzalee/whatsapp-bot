package plugins

import (
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types"
)

func init() {
	Register(&Command{
		Pattern:  "setprefix",
		IsSudo:   true,
		Category: "settings",
		Func: func(ctx *Context) error {
			if ctx.Text == "" {
				ctx.Reply(T().SetPrefixUsage)
				return nil
			}
			BotSettings.SetPrefixes(ctx.Text)
			if err := SaveSettings(); err != nil {
				ctx.Reply(fmt.Sprintf(T().SaveFailed, err.Error()))
				return err
			}
			display := strings.Join(BotSettings.GetPrefixes(), "  ")
			if display == "" {
				display = "(empty)"
			}
			ctx.Reply(fmt.Sprintf(T().SetPrefixUpdated, display))
			return nil
		},
	})

	Register(&Command{
		Pattern:  "setsudo",
		IsSudo:   true,
		Category: "settings",
		Func: func(ctx *Context) error {
			phone, lid := resolveSudoTarget(ctx, ctx.Text)
			if phone == "" && lid == "" {
				ctx.Reply(T().SetSudoUsage)
				return nil
			}
			if phone != "" {
				BotSettings.AddSudo(phone)
			}
			if lid != "" {
				BotSettings.AddSudo(lid)
			}
			if err := SaveSettings(); err != nil {
				ctx.Reply(fmt.Sprintf(T().SaveFailed, err.Error()))
				return err
			}
			display := phone
			if display == "" {
				display = lid
			}
			ctx.Reply(fmt.Sprintf(T().SudoAdded, display))
			return nil
		},
	})

	Register(&Command{
		Pattern:  "delsudo",
		IsSudo:   true,
		Category: "settings",
		Func: func(ctx *Context) error {
			phone, lid := resolveSudoTarget(ctx, ctx.Text)
			if phone == "" && lid == "" {
				ctx.Reply(T().DelSudoUsage)
				return nil
			}
			removed := false
			if phone != "" && BotSettings.RemoveSudo(phone) {
				removed = true
			}
			if lid != "" && BotSettings.RemoveSudo(lid) {
				removed = true
			}
			display := phone
			if display == "" {
				display = lid
			}
			if removed {
				_ = SaveSettings()
				ctx.Reply(fmt.Sprintf(T().SudoRemoved, display))
			} else {
				ctx.Reply(fmt.Sprintf(T().SudoNotFound, display))
			}
			return nil
		},
	})

	Register(&Command{
		Pattern:  "getsudo",
		IsSudo:   true,
		Category: "settings",
		Func: func(ctx *Context) error {
			BotSettings.mu.RLock()
			all := make([]string, len(BotSettings.Sudoers))
			copy(all, BotSettings.Sudoers)
			BotSettings.mu.RUnlock()

			// Only show phone-number entries; skip LID entries (those whose
			// reverse lookup returns a phone, meaning they ARE a LID).
			var phones []string
			for _, s := range all {
				if GetAltID(s+"@lid") == "" {
					// Can't resolve s as a LID → treat it as a phone.
					phones = append(phones, s)
				}
			}

			if len(phones) == 0 {
				ctx.Reply(T().SudoListEmpty)
				return nil
			}
			ctx.Reply(fmt.Sprintf(T().SudoList, strings.Join(phones, "\n")))
			return nil
		},
	})

	Register(&Command{
		Pattern:  "setmode",
		IsSudo:   true,
		Category: "settings",
		Func: func(ctx *Context) error {
			switch strings.ToLower(ctx.Text) {
			case "public":
				BotSettings.SetMode(ModePublic)
				_ = SaveSettings()
				ctx.Reply(T().ModePublicSet)
			case "private":
				BotSettings.SetMode(ModePrivate)
				_ = SaveSettings()
				ctx.Reply(T().ModePrivateSet)
			default:
				ctx.Reply(T().SetModeUsage)
			}
			return nil
		},
	})

	Register(&Command{
		Pattern:  "enablecmd",
		IsSudo:   true,
		Category: "settings",
		Func: func(ctx *Context) error {
			name := strings.ToLower(strings.TrimSpace(ctx.Text))
			if name == "" {
				ctx.Reply(T().EnableCmdUsage)
				return nil
			}
			if findCommand(name) == nil {
				ctx.Reply(fmt.Sprintf(T().CmdNotFound, name))
				return nil
			}
			BotSettings.EnableCmd(name)
			_ = SaveSettings()
			ctx.Reply(fmt.Sprintf(T().CmdEnabled, name))
			return nil
		},
	})

	Register(&Command{
		Pattern:  "disablecmd",
		IsSudo:   true,
		Category: "settings",
		Func: func(ctx *Context) error {
			name := strings.ToLower(strings.TrimSpace(ctx.Text))
			if name == "" {
				ctx.Reply(T().DisableCmdUsage)
				return nil
			}
			if findCommand(name) == nil {
				ctx.Reply(fmt.Sprintf(T().CmdNotFound, name))
				return nil
			}
			BotSettings.DisableCmd(name)
			_ = SaveSettings()
			ctx.Reply(fmt.Sprintf(T().CmdDisabledOK, name))
			return nil
		},
	})

	Register(&Command{
		Pattern:  "disablegc",
		IsSudo:   true,
		Category: "settings",
		Func: func(ctx *Context) error {
			if BotSettings.IsGCDisabled() {
				ctx.Reply(T().GCAlreadyDisabled)
				return nil
			}
			BotSettings.SetGCDisabled(true)
			_ = SaveSettings()
			ctx.Reply(T().GCDisabledSet)
			return nil
		},
	})

	Register(&Command{
		Pattern:  "enablegc",
		IsSudo:   true,
		Category: "settings",
		Func: func(ctx *Context) error {
			if !BotSettings.IsGCDisabled() {
				ctx.Reply(T().GCAlreadyEnabled)
				return nil
			}
			BotSettings.SetGCDisabled(false)
			_ = SaveSettings()
			ctx.Reply(T().GCEnabledSet)
			return nil
		},
	})

	Register(&Command{
		Pattern:  "ban",
		IsSudo:   true,
		Category: "settings",
		Func: func(ctx *Context) error {
			phone, lid := resolveSudoTarget(ctx, ctx.Text)
			if phone == "" && lid == "" {
				ctx.Reply(T().BanUsage)
				return nil
			}
			if phone != "" {
				BotSettings.BanUser(phone)
			}
			if lid != "" {
				BotSettings.BanUser(lid)
			}
			if err := SaveSettings(); err != nil {
				ctx.Reply(fmt.Sprintf(T().SaveFailed, err.Error()))
				return err
			}
			display := phone
			if display == "" {
				display = lid
			}
			ctx.Reply(fmt.Sprintf(T().UserBanned, display))
			return nil
		},
	})

	Register(&Command{
		Pattern:  "delban",
		IsSudo:   true,
		Category: "settings",
		Func: func(ctx *Context) error {
			phone, lid := resolveSudoTarget(ctx, ctx.Text)
			if phone == "" && lid == "" {
				ctx.Reply(T().DelBanUsage)
				return nil
			}
			removed := false
			if phone != "" && BotSettings.UnbanUser(phone) {
				removed = true
			}
			if lid != "" && BotSettings.UnbanUser(lid) {
				removed = true
			}
			display := phone
			if display == "" {
				display = lid
			}
			if removed {
				_ = SaveSettings()
				ctx.Reply(fmt.Sprintf(T().UserUnbanned, display))
			} else {
				ctx.Reply(fmt.Sprintf(T().UserNotBanned, display))
			}
			return nil
		},
	})

	Register(&Command{
		Pattern:  "getban",
		IsSudo:   true,
		Category: "settings",
		Func: func(ctx *Context) error {
			BotSettings.mu.RLock()
			all := make([]string, len(BotSettings.BannedUsers))
			copy(all, BotSettings.BannedUsers)
			BotSettings.mu.RUnlock()

			// Filter out LID entries — only show phone numbers.
			var phones []string
			for _, s := range all {
				if GetAltID(s+"@lid") == "" {
					phones = append(phones, s)
				}
			}

			if len(phones) == 0 {
				ctx.Reply(T().BanListEmpty)
				return nil
			}
			ctx.Reply(fmt.Sprintf(T().BanList, strings.Join(phones, "\n")))
			return nil
		},
	})
}

// resolveSudoTarget resolves the target for setsudo/delsudo/ban/delban.
// Priority:
//  1. DM with no arg → the chat partner.
//  2. arg provided → ResolveTarget (reply/mention/phone/LID).
func resolveSudoTarget(ctx *Context, arg string) (phone, lid string) {
	arg = strings.TrimSpace(arg)

	// Auto-target the DM chat partner when no argument is given.
	if arg == "" && !ctx.Event.Info.IsGroup {
		chat := ctx.Event.Info.Chat
		if chat.Server == types.HiddenUserServer {
			phone, lid = resolveJIDString(chat.String())
			if phone == "" && lid == "" {
				// LID not in store yet; use the raw LID user part so we can
				// still add it to sudoers before the mapping is saved.
				lid = chat.User
			}
		}
		return
	}

	return ResolveTarget(ctx, arg)
}

