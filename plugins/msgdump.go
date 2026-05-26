package plugins

import (
	"encoding/json"
	"fmt"
)

func init() {
	Register(&Command{
		Pattern:  "jsmsg",
		Aliases:  []string{"dump"},
		IsSudo:   true,
		Category: "owner",
		Func: func(ctx *Context) error {
			quoted := quotedMsg(ctx)
			var target interface{}
			if quoted != nil {
				target = quoted
			} else {
				target = ctx.Event.Message
			}
			out, err := json.MarshalIndent(target, "", "  ")
			if err != nil {
				ctx.Reply(fmt.Sprintf("Error: %s", err.Error()))
				return nil
			}
			result := string(out)
			if len(result) > 3500 {
				result = result[:3500] + "\n...(truncated)"
			}
			ctx.Reply("```" + result + "```")
			return nil
		},
	})
}
