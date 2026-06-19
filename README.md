# discord-go-router

`discord-go-router` is a small typed routing layer for
[`discordgo`](https://github.com/bwmarrin/discordgo). It lets you register
Discord application commands with Go structs, receive parsed arguments in a
typed context, and reply with messages, embeds, and simple buttons.

## Features

- Slash commands from struct fields and `dgr` tags
- Message and user context menu commands
- Typed command arguments through `Context[T]`
- Built-in interaction replies with ephemeral messages, embeds, and one button
- Choice fields using struct members of type `dgr.Choice`
- Discord resolved types for users, channels, roles, attachments, and mentionables

## Install

```sh
go get github.com/minumarapid/discord-go-router
```

## Quick Start

```go
package main

import (
	"log"
	"os"

	dgr "github.com/minumarapid/discord-go-router"
	"github.com/bwmarrin/discordgo"
)

type PingArgs struct {
	Text string `dgr:"text" desc:"Text to echo" required:"true"`
}

func main() {
	bot, err := dgr.New(os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	dgr.RegSlash(bot, "ping", "Reply with a pong", func(c *dgr.Context[PingArgs]) {
		embed := &discordgo.MessageEmbed{
			Title:       "Pong",
			Description: c.Args.Text,
			Color:       0x57F287,
		}

		if err := c.Reply("pong", false, nil, embed); err != nil {
			log.Println(err)
		}
	})

	if err := bot.Run(os.Getenv("DISCORD_GUILD_ID")); err != nil {
		log.Fatal(err)
	}
}
```

`Run` opens the Discord session, bulk-overwrites the registered application
commands for the given guild, and blocks until `Ctrl+C`.

## Slash Command Arguments

Define a struct for each slash command. Fields with a `dgr` tag become Discord
options.

```go
type Args struct {
	Name     string                         `dgr:"name" desc:"Display name" required:"true"`
	Count    int                            `dgr:"count" desc:"Repeat count"`
	Public   bool                           `dgr:"public" desc:"Show publicly"`
	User     *dgr.InteractionUser           `dgr:"user" desc:"Target user"`
	Channel  *discordgo.Channel             `dgr:"channel" desc:"Target channel"`
	Role     *discordgo.Role                `dgr:"role" desc:"Target role"`
	File     *discordgo.MessageAttachment   `dgr:"file" desc:"Attachment"`
	Mention  *dgr.Mentionable               `dgr:"mention" desc:"User or role"`
}
```

Supported field types:

| Go type | Discord option type |
| --- | --- |
| `string` | String |
| `int`, `int64` | Integer |
| `float32`, `float64` | Number |
| `bool` | Boolean |
| `*dgr.InteractionUser` | User |
| `*discordgo.Channel` | Channel |
| `*discordgo.Role` | Role |
| `*discordgo.MessageAttachment` | Attachment |
| `*dgr.Mentionable` | Mentionable |
| struct containing `dgr.Choice` fields | String choices |

Use `desc` for the Discord option description and `required:"true"` for
required options.

## Choices

Choices are represented as a nested struct whose fields are `dgr.Choice`.
The selected choice is set to `true`.

```go
type ColorChoices struct {
	Red   dgr.Choice
	Blue  dgr.Choice
	Green dgr.Choice
}

type PaintArgs struct {
	Color ColorChoices `dgr:"color" desc:"Paint color" required:"true"`
}

dgr.RegSlash(bot, "paint", "Pick a color", func(c *dgr.Context[PaintArgs]) {
	selected := dgr.Selected(&c.Args.Color)
	if selected == nil {
		_ = c.Reply("No color selected", true, nil)
		return
	}

	_ = c.Reply("Color selected", true, nil)
})
```

The Discord choice name and value are currently the Go field name.

## Replies

Use `Context.Reply` to respond to the interaction.

```go
err := c.Reply("hello", false, nil)
```

Signature:

```go
func (c *Context[T]) Reply(
	content string,
	ephemeral bool,
	button *Button[T],
	embeds ...*discordgo.MessageEmbed,
) error
```

Examples:

```go
_ = c.Reply("Only you can see this", true, nil)

embed := &discordgo.MessageEmbed{
	Title:       "Result",
	Description: "Done",
}
_ = c.Reply("", false, nil, embed)
```

## Buttons

Create a button from the current context, then pass it to `Reply`.
The button handler receives a new `Context[T]` with the same `Args`.

```go
dgr.RegSlash(bot, "confirm", "Show a confirm button", func(c *dgr.Context[struct{}]) {
	button, err := c.NewButton("Confirm", discordgo.PrimaryButton, func(c *dgr.Context[struct{}]) {
		_ = c.Reply("Confirmed", true, nil)
	})
	if err != nil {
		_ = c.Reply("Failed to create button", true, nil)
		return
	}

	_ = c.Reply("Continue?", true, button)
})
```

## Context Menu Commands

Message context menu command:

```go
dgr.RegMessageCtx(bot, "Inspect message", func(c *dgr.Context[discordgo.Message]) {
	_ = c.Reply(c.Args.Content, true, nil)
})
```

User context menu command:

```go
dgr.RegUserCtx(bot, "Inspect user", func(c *dgr.Context[discordgo.User]) {
	_ = c.Reply(c.Args.Username, true, nil)
})
```

## Error-Returning APIs

The `RegSlash`, `RegMessageCtx`, and `RegUserCtx` helpers are convenience
wrappers. Use the `E` variants when you want registration errors instead of a
panic or ignored error:

```go
err := dgr.RegSlashE(bot, "ping", "Reply with pong", handler)
err = dgr.RegMessageCtxE(bot, "Inspect message", messageHandler)
err = dgr.RegUserCtxE(bot, "Inspect user", userHandler)
```

## More Documentation

- [API reference](docs/api.md)
- [Command examples](docs/examples.md)
