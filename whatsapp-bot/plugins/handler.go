package plugins

import (
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// ModerationHook is called for every incoming message event.
type ModerationHook func(client *whatsmeow.Client, evt *events.Message)

var modHooks []ModerationHook

// RegisterModerationHook registers fn to run on every incoming message.
func RegisterModerationHook(fn ModerationHook) {
	modHooks = append(modHooks, fn)
}

// extractMsgText extracts the human-readable text from a message event.
func extractMsgText(evt *events.Message) string {
	return extractText(evt)
}

// NewHandler returns a whatsmeow event handler that drives the plugin system.
// Each message is handled in its own goroutine so the whatsmeow event loop is
// never blocked by command processing or network I/O.
func NewHandler(client *whatsmeow.Client) func(evt any) {
	return func(evt any) {
		switch v := evt.(type) {
		case *events.Message:
			go SaveUser(v)
			if v.Info.Sender.User == MetaJID.User {
				go HandleMetaAIResponse(client, v)
				return
			}
			for _, hook := range modHooks {
				h := hook
				go h(client, v)
			}
			go Dispatch(client, v)
		}
	}
}
