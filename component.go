package dgr

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/bwmarrin/discordgo"
)

type ButtonInvoker interface {
	Invoke(s *discordgo.Session, i *discordgo.Interaction)
}

type Button[T any] struct {
	Text     string
	CustomID string
	Style    discordgo.ButtonStyle
	Handler  func(c *Context[T])
	Args     T // 💡 コマンド実行時の引数をここに退避
}

func (b *Button[T]) Invoke(s *discordgo.Session, i *discordgo.Interaction) {
	newCtx := &Context[T]{
		Session:     s,
		Interaction: i,
		Args:        b.Args,
	}
	b.Handler(newCtx)
}

func (b *Button[T]) ToComponent() discordgo.MessageComponent {
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    b.Text,
				Style:    b.Style,
				CustomID: b.CustomID,
			},
		},
	}
}

func generateRandomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "btn_" + hex.EncodeToString(b)
}
