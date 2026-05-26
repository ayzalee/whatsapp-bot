package plugins

import (
	"fmt"
)

func init() {
	Register(&Command{
		Pattern:  "ping",
		Category: "utility",
		Func: func(ctx *Context) error {

			resp, err := ctx.ReplySync(T().Pong)
			if err != nil {
				return err
			}

			dt := resp.DebugTimings
			botTime := dt.Queue + dt.Marshal +
				dt.GetParticipants + dt.GetDevices +
				dt.GroupEncrypt + dt.PeerEncrypt +
				dt.Send
			ms := float64(botTime.Microseconds()) / 1000

			ctx.QueueEdit(resp.ID, fmt.Sprintf(T().PongLatency, ms))
			return nil
		},
	})
}
