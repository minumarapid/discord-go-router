# Examples

## Slash Command With Typed Arguments

```go
type SayArgs struct {
	Message string `dgr:"message" desc:"Message to send" required:"true"`
	Hidden  bool   `dgr:"hidden" desc:"Send as ephemeral response"`
}

dgr.RegSlash(bot, "say", "Echo a message", func(c *dgr.Context[SayArgs]) {
	opts := []dgr.ReplyOption{}
	if c.Args.Hidden {
		opts = append(opts, dgr.WithEphemeral())
	}

	_ = c.Reply(c.Args.Message, opts...)
})
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

## User, Role, Channel, And Attachment Options

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

## Button Response

```go
dgr.RegSlash(bot, "button", "Show a button", func(c *dgr.Context[struct{}]) {
	button, err := c.NewButton("Click me", discordgo.SuccessButton, func(c *dgr.Context[struct{}]) {
		_ = c.Reply("Clicked", dgr.WithEphemeral())
	})
	if err != nil {
		_ = c.Reply("Could not create button", dgr.WithEphemeral())
		return
	}

	_ = c.Reply("Press the button", dgr.WithEphemeral(), dgr.WithButton(button))
})
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
