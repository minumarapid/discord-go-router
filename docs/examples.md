# Examples

## Slash Command With Typed Arguments

Fields with a `dgr` tag become command options. `desc` sets the option
description and `required:"true"` marks it required. `string`, `int`/`int64`,
`float32`/`float64`, and `bool` map to the corresponding Discord option types.

```go
type SayArgs struct {
	Message string  `dgr:"message" desc:"Message to send" required:"true"`
	Count   int     `dgr:"count" desc:"Repeat count"`
	Rate    float64 `dgr:"rate" desc:"Rate"`
	Hidden  bool    `dgr:"hidden" desc:"Send as ephemeral response"`
}

dgr.RegSlash(bot, "say", "Echo a message", func(c *dgr.Context[SayArgs]) {
	opts := []dgr.ReplyOption{}
	if c.Args.Hidden {
		opts = append(opts, dgr.WithEphemeral())
	}

	_ = c.Reply(c.Args.Message, opts...)
})
```

The method form is equivalent and returns the registration error instead of
panicking:

```go
if err := bot.Slash("say", "Echo a message", func(c *dgr.Context[SayArgs]) {
	_ = c.Reply(c.Args.Message)
}); err != nil {
	log.Fatal(err)
}
```

## Embed Response

```go
dgr.RegSlash(bot, "status", "Show status", func(c *dgr.Context[struct{}]) {
	embed := &discordgo.MessageEmbed{
		Title:       "Status",
		Description: "All systems operational",
		Color:       0x57F287,
	}

	_ = c.Reply("", dgr.WithEmbeds(embed))
})
```

Use `struct{}` as the type argument for commands with no options.

## User, Role, Channel, And Attachment Options

Resolved option types are pointers and stay nil when omitted or unresolvable.

```go
type TargetArgs struct {
	User    *dgr.InteractionUser         `dgr:"user" desc:"User"`
	Role    *discordgo.Role              `dgr:"role" desc:"Role"`
	Channel *discordgo.Channel           `dgr:"channel" desc:"Channel"`
	File    *discordgo.MessageAttachment `dgr:"file" desc:"File"`
}

dgr.RegSlash(bot, "target", "Inspect resolved options", func(c *dgr.Context[TargetArgs]) {
	_ = c.Reply("Options parsed", dgr.WithEphemeral())
})
```

## Mentionable Option

A mentionable resolves to either a user or a role. Mark it `required:"true"`
when the handler dereferences it unconditionally.

```go
type MentionArgs struct {
	Target *dgr.Mentionable `dgr:"target" desc:"User or role" required:"true"`
}

dgr.RegSlash(bot, "mention", "Inspect a mentionable", func(c *dgr.Context[MentionArgs]) {
	switch c.Args.Target.Type {
	case dgr.MentionableTypeUser:
		_ = c.Reply("User selected", dgr.WithEphemeral())
	case dgr.MentionableTypeRole:
		_ = c.Reply("Role selected", dgr.WithEphemeral())
	}
})
```

## Tagged Choices

A struct field whose type is a struct containing `dgr.Choice` fields becomes a
String option with choices. The selected field is set to `true`.

```go
type ModeChoices struct {
	Fast dgr.Choice `name:"Fast mode" value:"fast"`
	Safe dgr.Choice `label:"Safe mode" value:"safe"`
	Auto dgr.Choice `dgr:"auto"`
}

type ModeArgs struct {
	Mode ModeChoices `dgr:"mode" desc:"Mode" required:"true"`
}

dgr.RegSlash(bot, "mode", "Select a mode", func(c *dgr.Context[ModeArgs]) {
	selected := dgr.SelectedChoiceOf(&c.Args.Mode)
	if selected == nil {
		_ = c.Reply("No mode selected", dgr.WithEphemeral())
		return
	}

	_ = c.Reply("Selected: "+selected.Value, dgr.WithEphemeral())
})
```

`dgr` sets both choice name and value, `name` (or its alias `label`) sets the
name, and `value` sets the value. Without tags the Go field name is used for
both. Use `dgr.Selected(&c.Args.Mode)` when only the `*dgr.Choice` pointer is
needed.

## Subcommands

```go
type KickArgs struct {
	User   *dgr.InteractionUser `dgr:"user" desc:"User to kick" required:"true"`
	Reason string               `dgr:"reason" desc:"Reason"`
}

moderation := dgr.Group(bot, "moderation", "Moderation commands")
dgr.RegSlash(moderation, "kick", "Kick a user", func(c *dgr.Context[KickArgs]) {
	_ = c.Reply("Kicked", dgr.WithEphemeral())
})
```

The method forms (`bot.Group`, `group.Slash`) are equivalent and return errors
instead of panicking.

## Subcommand Groups

```go
type BanArgs struct {
	User   *dgr.InteractionUser `dgr:"user" desc:"User to ban" required:"true"`
	Reason string               `dgr:"reason" desc:"Reason"`
}

admin := dgr.Group(bot, "admin", "Admin commands")
users := dgr.SubGroup(admin, "users", "User commands")
dgr.RegSlash(users, "ban", "Ban a user", func(c *dgr.Context[BanArgs]) {
	_ = c.Reply("Banned", dgr.WithEphemeral())
})
```

`admin.Group("users", "User commands")` is the same as
`dgr.SubGroup(admin, "users", "User commands")`.

## Button Response

`NewButton` copies the current `Args` into the button handler context.
`NewButtonRow` groups up to 5 buttons into one action row; buttons with
different type arguments can share a row.

```go
dgr.RegSlash(bot, "button", "Show a button", func(c *dgr.Context[struct{}]) {
	yes, err := c.NewButton("Yes", discordgo.SuccessButton, func(c *dgr.Context[struct{}]) {
		_ = c.Reply("Yes", dgr.WithEphemeral())
	})
	if err != nil {
		_ = c.Reply("Could not create button", dgr.WithEphemeral())
		return
	}

	no, err := c.NewButton("No", discordgo.DangerButton, func(c *dgr.Context[struct{}]) {
		_ = c.Reply("No", dgr.WithEphemeral())
	})
	if err != nil {
		_ = c.Reply("Could not create button", dgr.WithEphemeral())
		return
	}

	row, err := c.NewButtonRow(yes, no)
	if err != nil {
		_ = c.Reply("Could not create buttons", dgr.WithEphemeral())
		return
	}

	_ = c.Reply("Choose", dgr.WithEphemeral(), dgr.WithButtons(row))
})
```

For a single button, skip the row and use `WithButton`:

```go
dgr.RegSlash(bot, "confirm", "Show a confirm button", func(c *dgr.Context[struct{}]) {
	yes, err := c.NewButton("Confirm", discordgo.SuccessButton, func(c *dgr.Context[struct{}]) {
		_ = c.Reply("Confirmed", dgr.WithEphemeral())
	})
	if err != nil {
		_ = c.Reply("Could not create button", dgr.WithEphemeral())
		return
	}

	_ = c.Reply("Continue?", dgr.WithEphemeral(), dgr.WithButton(yes))
})
```

## Context Menu Commands

The target message or user is available as `c.Args`.

```go
dgr.RegMessageCtx(bot, "Inspect message", func(c *dgr.Context[discordgo.Message]) {
	_ = c.Reply(c.Args.Content, dgr.WithEphemeral())
})

dgr.RegUserCtx(bot, "Inspect user", func(c *dgr.Context[discordgo.User]) {
	_ = c.Reply(c.Args.Username, dgr.WithEphemeral())
})
```

## Message Create Handlers

`RegMsgCreate` subscribes to plain messages, not interactions. Pass channel IDs
to filter, or `"*"` to receive every channel. `MsgCreateCtx` has no `Reply`
helper; use `c.Session` to send messages.

```go
dgr.RegMsgCreate(bot, []string{"*"}, func(c *dgr.MsgCreateCtx) {
	_, _ = c.Session.ChannelMessageSend(c.Args.ChannelID, "Received: "+c.Args.Content)
})
```

To watch one channel only:

```go
if err := bot.MessageCreate([]string{"1234567890"}, func(c *dgr.MsgCreateCtx) {
	_, _ = c.Session.ChannelMessageSend(c.Args.ChannelID, "Noted")
}); err != nil {
	log.Fatal(err)
}
```

## Manual Session Lifecycle

Use `Run` for the common case. If you need to control the session lifecycle,
open the underlying Discord session before calling `SyncCommands`.

```go
if err := bot.Session.Open(); err != nil {
	log.Fatal(err)
}
defer bot.Stop()

if err := bot.SyncCommands(guildID); err != nil {
	log.Fatal(err)
}
```

Pass an empty `guildID` to sync global commands instead of guild commands.
