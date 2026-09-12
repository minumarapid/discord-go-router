# API Reference

This document summarizes the public surface of `discord-go-router`.

## Router

### `New`

```go
func New(token string) (*Dgr, error)
```

Creates a router and an underlying `discordgo.Session` using `"Bot " + token`.
The router registers its interaction handler and its message-create handler on
the session.

### `Run`

```go
func (d *Dgr) Run(guildID string) error
```

Opens the Discord session, calls `SyncCommands(guildID)`, and blocks until
`SIGINT`/`SIGTERM`. The session is closed before `Run` returns.

Sync failures inside `Run` are only logged (`Failed to sync commands: ...`).
`Run` itself returns `nil` in that case; it returns an error only when
`Session.Open` fails. An empty `guildID` syncs global commands.

### `SyncCommands`

```go
func (d *Dgr) SyncCommands(guildID string) error
```

Bulk-overwrites all registered commands with
`Session.ApplicationCommandBulkOverwrite`. An empty `guildID` means global
commands, a non-empty one means guild commands. The session must already be
open because the application ID is read from `Session.State.User.ID`; it
returns an error when the router, session, or session user is nil.

### `Stop`

```go
func (d *Dgr) Stop() error
```

Closes the underlying Discord session. Calling `Stop` on a nil router or nil
session is a no-op returning `nil`.

## Command Registration

`T` in the functions below must be a struct type. Use `struct{}` for commands
with no options. Only fields with a `dgr` tag become Discord options:

```go
type Args struct {
	Name string `dgr:"name" desc:"Display name" required:"true"`
}
```

- `dgr:"name"` sets the Discord option name (required for the field to be registered).
- `desc:"..."` sets the Discord option description.
- `required:"true"` marks the option as required.

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
| struct containing `dgr.Choice` fields | String with choices |

Other field types fall back to a String option without choices.

### Slash Commands

```go
type SlashTarget interface {
	// contains filtered or unexported methods
}

func RegSlash[T any](target SlashTarget, name string, description string, handler func(c *Context[T]))
func RegSlashE[T any](target SlashTarget, name string, description string, handler func(c *Context[T])) error

func (d *Dgr) Slash[T any](name string, description string, handler func(c *Context[T])) error
func (g *CommandGroup) Slash[T any](name string, description string, handler func(c *Context[T])) error
func (g *SubCommandGroup) Slash[T any](name string, description string, handler func(c *Context[T])) error
```

`target` must be `*Dgr` for a top-level slash command, `*CommandGroup` for a
subcommand, or `*SubCommandGroup` for a nested subcommand. The `Slash` methods
are shortcuts for `RegSlashE` with that target.

`RegSlashE` returns an error for a nil router/group, a nil handler, a
non-struct `T`, or a duplicate command/subcommand name. `RegSlash` panics on
any such error. The `Slash` methods return the error instead of panicking.

### Slash Command Groups

```go
func Group(d *Dgr, name string, description string) *CommandGroup
func GroupE(d *Dgr, name string, description string) (*CommandGroup, error)
func (d *Dgr) Group(name string, description string) (*CommandGroup, error)

func SubGroup(group *CommandGroup, name string, description string) *SubCommandGroup
func SubGroupE(group *CommandGroup, name string, description string) (*SubCommandGroup, error)
func (g *CommandGroup) Group(name string, description string) (*SubCommandGroup, error)
```

`Group` creates a top-level slash command whose options are subcommands or
subcommand groups. `SubGroup` creates a Discord subcommand group under a
`*CommandGroup`. `(*Dgr).Group` is the same as `GroupE`, and
`(*CommandGroup).Group` is the same as `SubGroupE` (note the method name is
`Group`, not `SubGroup`).

Register subcommands with `RegSlash`/`RegSlashE` or the `Slash` methods,
passing the returned `*CommandGroup` or `*SubCommandGroup` as the target. The
handler receives typed args parsed from the selected subcommand options.

`Group`, `SubGroup`, and `RegSlash` panic on registration errors (nil
router/group, duplicate name). The `E` variants and the `Slash`/`Group`
methods return errors instead.

### Message Context Menu Commands

```go
func RegMessageCtx(d *Dgr, name string, handler func(c *Context[discordgo.Message]))
func RegMessageCtxE(d *Dgr, name string, handler func(c *Context[discordgo.Message])) error
func (d *Dgr) Message(name string, handler func(c *Context[discordgo.Message])) error
```

Registers a message context menu command. The target message is available as
`c.Args` (zero `discordgo.Message` when unresolvable). `RegMessageCtxE` and
`(*Dgr).Message` return an error for a nil router or nil handler.
`RegMessageCtx` ignores that error (it does not panic).

### User Context Menu Commands

```go
func RegUserCtx(d *Dgr, name string, handler func(c *Context[discordgo.User]))
func RegUserCtxE(d *Dgr, name string, handler func(c *Context[discordgo.User])) error
func (d *Dgr) User(name string, handler func(c *Context[discordgo.User])) error
```

Registers a user context menu command. The target user is available as
`c.Args` (zero `discordgo.User` when unresolvable). `RegUserCtxE` and
`(*Dgr).User` return an error for a nil router or nil handler.
`RegUserCtx` ignores that error (it does not panic).

### Message Create Handlers

```go
func RegMsgCreate(d *Dgr, channelIDs []string, handler func(c *MsgCreateCtx))
func RegMsgCreateE(d *Dgr, channelIDs []string, handler func(c *MsgCreateCtx)) error
func (d *Dgr) MessageCreate(channelIDs []string, handler func(c *MsgCreateCtx)) error

type MsgCreateCtx struct {
	Session *discordgo.Session
	Args    *discordgo.Message
}
```

Registers plain message handlers (not interactions). One call can subscribe to
multiple channel IDs, and multiple handlers can subscribe to the same channel.
The special channel ID `"*"` matches every channel: on each message, handlers
registered for that `ChannelID` run first, then handlers registered for `"*"`.

`RegMsgCreateE` and `(*Dgr).MessageCreate` return an error for a nil router or
nil handler. `RegMsgCreate` ignores that error (it does not panic).
`MsgCreateCtx` has no `Reply` helper; send messages via `c.Session`, e.g.
`c.Session.ChannelMessageSend(c.Args.ChannelID, "hello")`. `c.Args` is the
`discordgo.MessageCreate.Message` payload.

## Context

```go
type Context[T any] struct {
	Session     *discordgo.Session
	Interaction *discordgo.Interaction
	Args        T
}
```

`Context` is passed to slash, context menu, and button handlers. `Args`
contains the parsed slash command options, the context menu target, or (for
buttons) a copy of the `Args` from when the button was created.

### `Reply`

```go
func (c *Context[T]) Reply(content string, opts ...ReplyOption) error
```

Sends the initial interaction response with type
`InteractionResponseChannelMessageWithSource`. It returns an error when the
context, session, or interaction is nil, or when Discord rejects the response.
Each interaction accepts only one initial `Reply`; follow-up messages must use
the session directly (e.g. `Session.FollowupMessageCreate`).

### Reply Options

```go
type ReplyOption func(*discordgo.InteractionResponseData)

func WithEphemeral() ReplyOption
func WithButton[T any](button *Button[T]) ReplyOption
func WithButtons(row *ButtonRow) ReplyOption
func WithEmbeds(embeds ...*discordgo.MessageEmbed) ReplyOption
```

- `WithEphemeral` sets the ephemeral message flag.
- `WithButton` appends the button wrapped in its own single-button action row.
- `WithButtons` appends a row built with `NewButtonRow`, so up to 5 buttons
  render horizontally. Multiple `WithButton`/`WithButtons` options append
  multiple rows.
- `WithEmbeds` appends embeds to the response.

Nil buttons/rows are ignored.

### `NewButton` / `NewButtonRow`

```go
func (c *Context[T]) NewButton(text string, style discordgo.ButtonStyle, handler func(c *Context[T])) (*Button[T], error)
func (c *Context[T]) NewButtonRow(buttons ...ButtonComponent) (*ButtonRow, error)
```

`NewButton` creates a button with a generated `btn_<hex>` custom ID, registers
it in the router's button pool, and copies the current `Args` into the button
so the button handler receives them. It returns an error when the context has
no router or when the handler is nil.

`NewButtonRow` groups buttons into one Discord action row. It needs no router,
drops nil entries, and returns an error when given more than 5 buttons. Buttons
with different generic argument types can share a row because the row stores
`ButtonComponent` values.

## Buttons

```go
type ButtonComponent interface {
	ToButtonComponent() discordgo.Button
}

type Button[T any] struct {
	Text     string
	CustomID string
	Style    discordgo.ButtonStyle
	Handler  func(c *Context[T])
	Args     T
}

type ButtonRow struct {
	Buttons []ButtonComponent
}
```

`*Button[T]` implements `ButtonComponent`. `ToButtonComponent` converts one
button to `discordgo.Button`; `(*Button[T]).ToComponent` and
`(*ButtonRow).ToComponent` convert to a `discordgo.ActionsRow` for
`WithButton`/`WithButtons`.

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

### `Selected` / `SelectedChoiceOf`

```go
type SelectedChoice struct {
	Choice *Choice
	Name   string
	Value  string
}

func Selected[T any](structPtr *T) *Choice
func SelectedChoiceOf[T any](structPtr *T) *SelectedChoice
```

`Selected` returns a pointer to the selected `Choice` field in a choice struct,
or `nil` when no choice is selected (or the pointer is nil/not a struct).
`SelectedChoiceOf` returns the selected choice plus its configured `Name` and
`Value` from the tags, or `nil` under the same conditions.

### `InteractionUser`

```go
type InteractionUser struct {
	*discordgo.User
	*discordgo.Member
}
```

Represents a resolved user option. Discord may include user data, member data,
or both. The field stays nil when the option was omitted or unresolvable.

### `Mentionable`

```go
type Mentionable struct {
	User *InteractionUser
	Role *discordgo.Role
	Type MentionableType
}

type MentionableType string

const (
	MentionableTypeUser MentionableType = "USER"
	MentionableTypeRole MentionableType = "ROLE"
)
```

Represents a resolved mentionable option. `Type` is either
`MentionableTypeUser` (with `User` set) or `MentionableTypeRole` (with `Role`
set). The `*Mentionable` field stays nil when the option was omitted or
unresolvable.
