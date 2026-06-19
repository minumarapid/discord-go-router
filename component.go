package dgr

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type ButtonInvoker interface {
	Invoke(s *discordgo.Session, i *discordgo.Interaction)
}

type ButtonComponent interface {
	ToButtonComponent() discordgo.Button
}

type Button[T any] struct {
	Text     string
	CustomID string
	Style    discordgo.ButtonStyle
	Handler  func(c *Context[T])
	Args     T
	dgr      *Dgr
}

type ButtonRow struct {
	Buttons []ButtonComponent
}

func (b *Button[T]) Invoke(s *discordgo.Session, i *discordgo.Interaction) {
	if b == nil || b.Handler == nil {
		return
	}
	newCtx := &Context[T]{
		Session:     s,
		Interaction: i,
		Args:        b.Args,
		dgr:         b.dgr,
	}
	b.Handler(newCtx)
}

func (b *Button[T]) ToButtonComponent() discordgo.Button {
	if b == nil {
		return discordgo.Button{}
	}
	return discordgo.Button{
		Label:    b.Text,
		Style:    b.Style,
		CustomID: b.CustomID,
	}
}

func (b *Button[T]) ToComponent() discordgo.MessageComponent {
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			b.ToButtonComponent(),
		},
	}
}

func (r *ButtonRow) ToComponent() discordgo.MessageComponent {
	if r == nil {
		return discordgo.ActionsRow{}
	}

	components := make([]discordgo.MessageComponent, 0, len(r.Buttons))
	for _, button := range r.Buttons {
		if button == nil {
			continue
		}
		components = append(components, button.ToButtonComponent())
	}

	return discordgo.ActionsRow{
		Components: components,
	}
}

func generateRandomID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random button ID: %w", err)
	}
	return "btn_" + hex.EncodeToString(b), nil
}
