# API Reference

This document summarizes the public surface of `discord-go-router`.

## Router

### `New`

```go
func New(token string) (*Dgr, error)
```

Creates a router and an underlying `discordgo.Session` using `Bot ` plus the
given token. The router registers its interaction handler on the session.

### `Run`

```go
func (d *Dgr) Run(guildID string) error
```

Opens the Discord session, syncs registered commands to `guildID`, and waits
for an interrupt signal. The session is closed before `Run` returns.

### `SyncCommands`

```go
func (d *Dgr) SyncCommands(guildID string) error
```

Bulk-overwrites all registered commands for the given guild. The session must
already be open because the application ID is read from `Session.State.User.ID`.

### `Stop`

```go
func (d *Dgr) Stop() error
```

Closes the underlying Discord session. Calling `Stop` on a nil router or nil
session is a no-op.

## Command Registration

### Slash Commands

```go
type SlashTarget interface {
	// contains filtered or unexported methods
}

func RegSlash[T any](target SlashTarget, name string, description string, handler func(c *Context[T]))
func RegSlashE[T any](target SlashTarget, name string, description string, handler func(c *Context[T])) error
```

`target` must be `*Dgr` for a top-level slash command, `*CommandGroup` for a
subcommand, or `*SubCommandGroup` for a nested subcommand. `T` must be a struct.
Fields with a `dgr` tag become slash command options. `RegSlash` panics when
registration fails. `RegSlashE` returns the error.

### Slash Command Groups

```go
func Group(d *Dgr, name string, description string) *CommandGroup
func GroupE(d *Dgr, name string, description string) (*CommandGroup, error)
func SubGroup(group *CommandGroup, name string, description string) *SubCommandGroup
func SubGroupE(group *CommandGroup, name string, description string) (*SubCommandGroup, error)
```

`Group` creates a top-level slash command whose options are subcommands or
subcommand groups. `SubGroup` creates a Discord subcommand group under a
`*CommandGroup`. Register subcommands with `RegSlash` or `RegSlashE`, passing
the returned `*CommandGroup` or `*SubCommandGroup` as the first argument. The
handler receives typed args parsed from the selected subcommand options.
`Group`, `SubGroup`, and `RegSlash` panic on registration errors. The `E`
variants return errors.

### Message Context Menu Commands

```go
func RegMessageCtx(d *Dgr, name string, handler func(c *Context[discordgo.Message]))
func RegMessageCtxE(d *Dgr, name string, handler func(c *Context[discordgo.Message])) error
```

Registers a message context menu command. The target message is available as
`c.Args`.

### User Context Menu Commands

```go
func RegUserCtx(d *Dgr, name string, handler func(c *Context[discordgo.User]))
func RegUserCtxE(d *Dgr, name string, handler func(c *Context[discordgo.User])) error
```

Registers a user context menu command. The target user is available as
`c.Args`.

## Context

```go
type Context[T any] struct {
	Session     *discordgo.Session
	Interaction *discordgo.Interaction
	Args        T
}
```

`Context` is passed to registered handlers. `Args` contains the parsed slash
command options or context menu target.

### `Reply`

```go
func (c *Context[T]) Reply(content string, opts ...ReplyOption) error
```

Sends an interaction response. Use `WithEphemeral`, `WithButton`, and
`WithEmbeds` to configure the response.

### Reply Options

```go
type ReplyOption func(*discordgo.InteractionResponseData)

func WithEphemeral() ReplyOption
func WithButton[T any](button *Button[T]) ReplyOption
func WithEmbeds(embeds ...*discordgo.MessageEmbed) ReplyOption
```

### `NewButton`

```go
func (c *Context[T]) NewButton(text string, style discordgo.ButtonStyle, handler func(c *Context[T])) (*Button[T], error)
```

Creates and registers a button tied to the router. The handler receives the same
generic argument type as the original context, and the current `Args` value is
copied into the button handler context.

## Types

### `Choice`

```go
type Choice bool
```

Use `Choice` fields inside a nested struct to define string choices for a slash
command option. The chosen field is set to `true`. Choice fields can customize
their Discord choice metadata with tags:

```go
type ColorChoices struct {
	Red   dgr.Choice `name:"Red" value:"red"`
	Green dgr.Choice `dgr:"green"`
}
```

`dgr` sets both name and value, `name` sets the Discord choice name, `label` is
an alias for `name`, and `value` sets the Discord choice value. Without tags,
the Go field name is used for both name and value.

### `Selected`

```go
func Selected[T any](structPtr *T) *Choice
```

Returns a pointer to the selected `Choice` field in a choice struct, or `nil`
when no choice is selected.

### `SelectedChoiceOf`

```go
func SelectedChoiceOf[T any](structPtr *T) *SelectedChoice
```

Returns the selected choice with its configured `Name` and `Value`, or `nil`
when no choice is selected.

### `InteractionUser`

```go
type InteractionUser struct {
	*discordgo.User
	*discordgo.Member
}
```

Represents a resolved user option. Discord may include user data, member data,
or both.

### `Mentionable`

```go
type Mentionable struct {
	User *InteractionUser
	Role *discordgo.Role
	Type MentionableType
}
```

Represents a resolved mentionable option. `Type` is either
`MentionableTypeUser` or `MentionableTypeRole`.
