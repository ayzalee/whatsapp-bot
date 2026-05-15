package plugins

import (
	"fmt"
)

func init() {
	Register(&Command{
		Pattern:  "ping",
		Category: "utility",
		Func: func(ctx *Context) error {
			// ReplySync waits for the server ACK so we get DebugTimings.
			resp, err := ctx.ReplySync(T().Pong)
			if err != nil {
				return err
			}

			// "Bot latency" = everything the bot did before the server ACK:
			//   dispatch → lock wait → marshal → encrypt → socket write
			// This is the same thing Baileys measures when it resolves
			// sendMessage() after the WebSocket write (not after the ACK).
			dt := resp.DebugTimings
			botTime := dt.Queue + dt.Marshal +
				dt.GetParticipants + dt.GetDevices +
				dt.GroupEncrypt + dt.PeerEncrypt +
				dt.Send
			ms := float64(botTime.Microseconds()) / 1000

			// Edit to show the real number (typically 2–15 ms on warm sessions).
			ctx.QueueEdit(resp.ID, fmt.Sprintf(T().PongLatency, ms))
			return nil
		},
	})
}
