package dgr

import "github.com/bwmarrin/discordgo"

type Context[T any] struct {
	Session     *discordgo.Session
	Interaction *discordgo.Interaction
	Args        T
	dgr         *Dgr
}

func (c *Context[T]) Reply(content string, ephemeral bool, button *Button[T]) {
	var flags discordgo.MessageFlags
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}

	var components []discordgo.MessageComponent
	if button != nil {
		components = append(components, button.ToComponent()) // 💡 コンポーネントに変換
	}

	c.Session.InteractionRespond(c.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Flags:      flags,
			Components: components,
		},
	})
}

func (c *Context[T]) NewButton(text string, style discordgo.ButtonStyle, handler func(c *Context[T])) *Button[T] {
	customID := generateRandomID()

	btn := &Button[T]{
		Text:     text,
		CustomID: customID,
		Style:    style,
		Handler:  handler,
		Args:     c.Args,
	}

	c.dgr.buttonPool[customID] = btn

	return btn
}
