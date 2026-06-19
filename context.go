package dgr

import (
	"errors"

	"github.com/bwmarrin/discordgo"
)

type Context[T any] struct {
	Session     *discordgo.Session
	Interaction *discordgo.Interaction
	Args        T
	dgr         *Dgr
}

func (c *Context[T]) Reply(content string, ephemeral bool, button *Button[T], embeds ...*discordgo.MessageEmbed) error {
	if c == nil || c.Session == nil || c.Interaction == nil {
		return errors.New("dgr: nil context, session, or interaction")
	}

	var flags discordgo.MessageFlags
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}

	var components []discordgo.MessageComponent
	if button != nil {
		components = append(components, button.ToComponent())
	}

	return c.Session.InteractionRespond(c.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Flags:      flags,
			Components: components,
			Embeds:     embeds,
		},
	})
}

func (c *Context[T]) NewButton(text string, style discordgo.ButtonStyle, handler func(c *Context[T])) (*Button[T], error) {
	if c == nil || c.dgr == nil {
		return nil, errors.New("dgr: nil router in context")
	}
	if handler == nil {
		return nil, errors.New("dgr: nil button handler")
	}

	customID, err := generateRandomID()
	if err != nil {
		return nil, err
	}

	btn := &Button[T]{
		Text:     text,
		CustomID: customID,
		Style:    style,
		Handler:  handler,
		Args:     c.Args,
		dgr:      c.dgr,
	}

	c.dgr.mu.Lock()
	c.dgr.initLocked()
	c.dgr.buttonPool[customID] = btn
	c.dgr.mu.Unlock()

	return btn, nil
}
