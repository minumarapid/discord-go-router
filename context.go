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

type ReplyOption func(*discordgo.InteractionResponseData)

func WithEphemeral() ReplyOption {
	return func(data *discordgo.InteractionResponseData) {
		data.Flags |= discordgo.MessageFlagsEphemeral
	}
}

func WithButton[T any](button *Button[T]) ReplyOption {
	return func(data *discordgo.InteractionResponseData) {
		if button != nil {
			data.Components = append(data.Components, button.ToComponent())
		}
	}
}

func WithEmbeds(embeds ...*discordgo.MessageEmbed) ReplyOption {
	return func(data *discordgo.InteractionResponseData) {
		data.Embeds = append(data.Embeds, embeds...)
	}
}

func (c *Context[T]) Reply(content string, opts ...ReplyOption) error {
	if c == nil || c.Session == nil || c.Interaction == nil {
		return errors.New("dgr: nil context, session, or interaction")
	}

	data := &discordgo.InteractionResponseData{
		Content: content,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(data)
		}
	}

	return c.Session.InteractionRespond(c.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
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
