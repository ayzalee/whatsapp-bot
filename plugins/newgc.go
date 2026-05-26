package plugins

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/jpeg"
	"io"
	"net/http"
	"regexp"
	"strings"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

var ogImageRe = regexp.MustCompile(`(?i)<meta[^>]+property=["']og:image["'][^>]+content=["']([^"']+)["']|<meta[^>]+content=["']([^"']+)["'][^>]+property=["']og:image["']`)

func fetchOGImage(url string) ([]byte, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	m := ogImageRe.FindSubmatch(body)
	if m == nil {
		return nil, "", fmt.Errorf("og:image not found")
	}

	imgURL := string(m[1])
	if imgURL == "" {
		imgURL = string(m[2])
	}

	imgResp, err := http.Get(imgURL)
	if err != nil {
		return nil, "", err
	}
	defer imgResp.Body.Close()

	imgData, err := io.ReadAll(imgResp.Body)
	if err != nil {
		return nil, "", err
	}

	return imgData, imgURL, nil
}

func toJPEG(data []byte) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()

	const maxDim = 640
	dw, dh := sw, sh

	if sw > maxDim || sh > maxDim {
		if sw >= sh {
			dh = sh * maxDim / sw
			dw = maxDim
		} else {
			dw = sw * maxDim / sh
			dh = maxDim
		}
	}

	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))

	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			srcX := float64(x) * float64(sw-1) / float64(dw-1)
			srcY := float64(y) * float64(sh-1) / float64(dh-1)

			x0, y0 := int(srcX), int(srcY)
			x1, y1 := x0+1, y0+1

			if x1 >= sw {
				x1 = sw - 1
			}
			if y1 >= sh {
				y1 = sh - 1
			}

			fx, fy := srcX-float64(x0), srcY-float64(y0)

			r00, g00, b00, a00 := src.At(b.Min.X+x0, b.Min.Y+y0).RGBA()
			r10, g10, b10, a10 := src.At(b.Min.X+x1, b.Min.Y+y0).RGBA()
			r01, g01, b01, a01 := src.At(b.Min.X+x0, b.Min.Y+y1).RGBA()
			r11, g11, b11, a11 := src.At(b.Min.X+x1, b.Min.Y+y1).RGBA()

			blend := func(v00, v10, v01, v11 uint32) uint8 {
				top := float64(v00)*(1-fx) + float64(v10)*fx
				bot := float64(v01)*(1-fx) + float64(v11)*fx
				return uint8((top*(1-fy) + bot*fy) / 257)
			}

			dst.SetRGBA(x, y, color.RGBA{
				R: blend(r00, r10, r01, r11),
				G: blend(g00, g10, g01, g11),
				B: blend(b00, b10, b01, b11),
				A: blend(a00, a10, a01, a11),
			})
		}
	}

	var buf bytes.Buffer

	if err := jpeg.Encode(&buf, dst, &jpeg.Options{
		Quality: 85,
	}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func init() {
	Register(&Command{
		Pattern:  "newgc",
		Category: "group",
		Func: func(ctx *Context) error {

			name := ctx.Text

			if name == "" {
				ctx.Reply(menuHeader("newgc") + T().NewGCUsage)
				return nil
			}

			if len([]rune(name)) > 25 {
				ctx.Reply(T().NewGCNameTooLong)
				return nil
			}

			resp, err := ctx.ReplySync(T().NewGCCreating)
			if err != nil {
				return err
			}

			loaderID := resp.ID

			groupInfo, err := ctx.Client.CreateGroup(
				context.Background(),
				whatsmeow.ReqCreateGroup{
					Name: name,
				},
			)

			if err != nil {
				ctx.QueueEdit(
					loaderID,
					fmt.Sprintf(T().NewGCFailed, err.Error()),
				)
				return nil
			}

			ctx.QueueEdit(loaderID, T().NewGCSettingDesc)

			desc := `• _*Welcome آپ  Ka ہـمــٓـــارے  Group مـیـٓـــں - 🥂🫀*_

• _*Group کـــّے  Rules جــٓــو  Allowed ہـیــٔـں - ✅🍷*_

• _*Aur کــٓـچـھ  Rules جــٓــو  Not Allowed ہـیــٔں - ❌🍸*_

• _*Group میں  Sab مـمـبــٓـرز  Ki عـــٓــزت  Krni ہــٓـے  - 🫠♥️*_

• _*Group میــٓـں  Link بـھـیـجـنــٓـا  Allowed نـہـیــٓـں  - 💀♥️*_

• _*Yahan پــٓـر  Gali وغـیــٓـرہ  Allowed نـہـیــٓـں  - ⚠️👀*_

• _*Aur کــوئــَـی  Gandi فــوٹـــٓـو  Ya ویــڈیـــٓـو  Allowed نـہـیــٓـں  - 🔞⚠️*_

• _*Without اجــٓـازت  Kisi کــٓـے  InBox جـانــٓے  Ki اجــٓـازت  Nahi - ⚡*_

• _*Jo گــٓـروپ  Ke اصــٓـول  Tore گــٓـا  Usko بـغـیــٓـر  Kisi وارنـنــگ  Ke گــٓـروپ  Se نـکــٓـال  Diya جـائــٓـے  Ga  - 🙂💔*_

• _*Ya گــٓـروپ  Sirf بـــٓـات چـیـت  Ke لـیــٓـے  Bnaya گـیــٓـا  Hai - 🫠♥️*_

• _*Group کــٓـا  Link نـیـچــٓـے  Diya گیا  Hai - ⬇️🙂*_`

			_ = ctx.Client.SetGroupDescription(
				context.Background(),
				groupInfo.JID,
				desc,
			)

			ctx.QueueEdit(loaderID, T().NewGCFetchingIcon)

			imgData, _, err := fetchOGImage(
				"https://github.com/zaelix/whatsapp-bot",
			)

			if err == nil {
				jpegData, err := toJPEG(imgData)

				if err == nil {
					_, _ = ctx.Client.SetGroupPhoto(
						context.Background(),
						groupInfo.JID,
						jpegData,
					)
				}
			}

			ctx.QueueEdit(loaderID, T().NewGCFetchingLink)

			inviteLink, err := ctx.Client.GetGroupInviteLink(
				context.Background(),
				groupInfo.JID,
				false,
			)

			if err != nil {
				ctx.QueueEdit(
					loaderID,
					fmt.Sprintf(
						T().NewGCDone,
						groupInfo.JID.String(),
						"",
					),
				)
				return nil
			}

			previewURL := "https://chat.whatsapp.com/" +
				strings.TrimPrefix(
					inviteLink,
					"https://chat.whatsapp.com/",
				)

			thumbData, _, ferr := fetchOGImage(
				"https://github.com/zaelix/whatsapp-bot",
			)

			var thumbJPEG []byte

			if ferr == nil {
				thumbJPEG, _ = toJPEG(thumbData)
			}

			msg := &waProto.Message{
				ExtendedTextMessage: &waProto.ExtendedTextMessage{
					Text: proto.String(
						fmt.Sprintf(
							T().NewGCDone,
							name,
							previewURL,
						),
					),
					Title:         proto.String(name),
					Description:   proto.String(desc),
					MatchedText:   proto.String(previewURL),
					JPEGThumbnail: thumbJPEG,
				},
			}

			editMsg := ctx.Client.BuildEdit(
				ctx.Event.Info.Chat,
				loaderID,
				msg,
			)

			sendQueue <- sendTask{
				client: ctx.Client,
				to:     ctx.Event.Info.Chat,
				msg:    editMsg,
				id:     ctx.Client.GenerateMessageID(),
			}

			return nil
		},
	})
}
