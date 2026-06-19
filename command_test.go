package dgr

import (
	"reflect"
	"testing"

	"github.com/bwmarrin/discordgo"
)

type testModeChoices struct {
	Fast  Choice `name:"Fast mode" value:"fast"`
	Safe  Choice `label:"Safe mode" value:"safe"`
	Auto  Choice `dgr:"auto"`
	Plain Choice
}

func TestRegSlashChoiceTags(t *testing.T) {
	type args struct {
		Mode testModeChoices `dgr:"mode" desc:"Mode" required:"true"`
	}

	d := &Dgr{}
	if err := RegSlashE(d, "mode", "Select a mode", func(c *Context[args]) {}); err != nil {
		t.Fatal(err)
	}

	if len(d.commands) != 1 || len(d.commands[0].Options) != 1 {
		t.Fatalf("unexpected command registration: %#v", d.commands)
	}

	choices := d.commands[0].Options[0].Choices
	want := map[string]string{
		"Fast mode": "fast",
		"Safe mode": "safe",
		"auto":      "auto",
		"Plain":     "Plain",
	}

	if len(choices) != len(want) {
		t.Fatalf("got %d choices, want %d", len(choices), len(want))
	}

	for _, choice := range choices {
		value, ok := choice.Value.(string)
		if !ok {
			t.Fatalf("choice %q value is %T, want string", choice.Name, choice.Value)
		}
		if want[choice.Name] != value {
			t.Fatalf("choice %q value = %q, want %q", choice.Name, value, want[choice.Name])
		}
	}
}

func TestSetChoiceValueUsesTaggedValue(t *testing.T) {
	var choices testModeChoices
	opt := &discordgo.ApplicationCommandInteractionDataOption{
		Type:  discordgo.ApplicationCommandOptionString,
		Name:  "mode",
		Value: "safe",
	}

	setScalarOrChoiceValue(reflect.ValueOf(&choices).Elem(), opt)

	if !bool(choices.Safe) {
		t.Fatal("Safe choice was not selected")
	}
	if choices.Fast || choices.Auto || choices.Plain {
		t.Fatalf("unexpected selected choices: %#v", choices)
	}
}

func TestSelectedChoiceOfReturnsTags(t *testing.T) {
	choices := testModeChoices{Fast: true}

	selected := SelectedChoiceOf(&choices)
	if selected == nil {
		t.Fatal("selected choice is nil")
	}
	if selected.Name != "Fast mode" || selected.Value != "fast" {
		t.Fatalf("selected = %#v, want name %q and value %q", selected, "Fast mode", "fast")
	}
	if selected.Choice != &choices.Fast {
		t.Fatal("selected choice pointer does not point to the selected field")
	}
}

func TestGroupRegistersSubcommand(t *testing.T) {
	type args struct {
		Name string `dgr:"name" desc:"Name" required:"true"`
	}

	d := &Dgr{}
	group, err := GroupE(d, "admin", "Admin commands")
	if err != nil {
		t.Fatal(err)
	}
	if err := RegSlashE(group, "ban", "Ban a user", func(c *Context[args]) {}); err != nil {
		t.Fatal(err)
	}

	if len(d.commands) != 1 {
		t.Fatalf("got %d commands, want 1", len(d.commands))
	}

	command := d.commands[0]
	if command.Name != "admin" || command.Description != "Admin commands" {
		t.Fatalf("unexpected command: %#v", command)
	}
	if len(command.Options) != 1 {
		t.Fatalf("got %d command options, want 1", len(command.Options))
	}

	subcommand := command.Options[0]
	if subcommand.Type != discordgo.ApplicationCommandOptionSubCommand {
		t.Fatalf("subcommand type = %v, want %v", subcommand.Type, discordgo.ApplicationCommandOptionSubCommand)
	}
	if subcommand.Name != "ban" || subcommand.Description != "Ban a user" {
		t.Fatalf("unexpected subcommand: %#v", subcommand)
	}
	if len(subcommand.Options) != 1 || subcommand.Options[0].Name != "name" {
		t.Fatalf("unexpected subcommand options: %#v", subcommand.Options)
	}
	if d.interactionHandlers[groupHandlerKey("admin", "ban")] == nil {
		t.Fatal("subcommand handler was not registered")
	}
}

func TestParseSlashArgsFromSubcommandOptions(t *testing.T) {
	type args struct {
		Name string `dgr:"name" desc:"Name" required:"true"`
	}

	parsed := parseSlashArgsFromOptions[args](
		[]*discordgo.ApplicationCommandInteractionDataOption{
			{
				Type:  discordgo.ApplicationCommandOptionString,
				Name:  "name",
				Value: "hina",
			},
		},
		nil,
	)

	if parsed.Name != "hina" {
		t.Fatalf("Name = %q, want %q", parsed.Name, "hina")
	}
}

func TestSubGroupRegistersSubcommand(t *testing.T) {
	type args struct {
		Name string `dgr:"name" desc:"Name" required:"true"`
	}

	d := &Dgr{}
	group, err := GroupE(d, "admin", "Admin commands")
	if err != nil {
		t.Fatal(err)
	}
	users, err := SubGroupE(group, "users", "User commands")
	if err != nil {
		t.Fatal(err)
	}
	if err := RegSlashE(users, "ban", "Ban a user", func(c *Context[args]) {}); err != nil {
		t.Fatal(err)
	}

	command := d.commands[0]
	if len(command.Options) != 1 {
		t.Fatalf("got %d command options, want 1", len(command.Options))
	}

	subGroup := command.Options[0]
	if subGroup.Type != discordgo.ApplicationCommandOptionSubCommandGroup {
		t.Fatalf("subgroup type = %v, want %v", subGroup.Type, discordgo.ApplicationCommandOptionSubCommandGroup)
	}
	if subGroup.Name != "users" || subGroup.Description != "User commands" {
		t.Fatalf("unexpected subgroup: %#v", subGroup)
	}
	if len(subGroup.Options) != 1 {
		t.Fatalf("got %d subgroup options, want 1", len(subGroup.Options))
	}

	subcommand := subGroup.Options[0]
	if subcommand.Type != discordgo.ApplicationCommandOptionSubCommand {
		t.Fatalf("subcommand type = %v, want %v", subcommand.Type, discordgo.ApplicationCommandOptionSubCommand)
	}
	if subcommand.Name != "ban" || subcommand.Description != "Ban a user" {
		t.Fatalf("unexpected subcommand: %#v", subcommand)
	}
	if len(subcommand.Options) != 1 || subcommand.Options[0].Name != "name" {
		t.Fatalf("unexpected subcommand options: %#v", subcommand.Options)
	}
	if d.interactionHandlers[subGroupHandlerKey("admin", "users", "ban")] == nil {
		t.Fatal("subcommand handler was not registered")
	}
}

func TestInteractionHandlerKeyForSubGroup(t *testing.T) {
	opt := &discordgo.ApplicationCommandInteractionDataOption{
		Type: discordgo.ApplicationCommandOptionSubCommandGroup,
		Name: "users",
		Options: []*discordgo.ApplicationCommandInteractionDataOption{
			{
				Type: discordgo.ApplicationCommandOptionSubCommand,
				Name: "ban",
			},
		},
	}

	got := interactionHandlerKey("admin", opt)
	want := subGroupHandlerKey("admin", "users", "ban")
	if got != want {
		t.Fatalf("key = %q, want %q", got, want)
	}
}

func TestSelectedSubcommandOptionForSubGroup(t *testing.T) {
	subcommand := &discordgo.ApplicationCommandInteractionDataOption{
		Type: discordgo.ApplicationCommandOptionSubCommand,
		Name: "ban",
		Options: []*discordgo.ApplicationCommandInteractionDataOption{
			{
				Type:  discordgo.ApplicationCommandOptionString,
				Name:  "name",
				Value: "hina",
			},
		},
	}

	got := selectedSubcommandOption([]*discordgo.ApplicationCommandInteractionDataOption{
		{
			Type:    discordgo.ApplicationCommandOptionSubCommandGroup,
			Name:    "users",
			Options: []*discordgo.ApplicationCommandInteractionDataOption{subcommand},
		},
	})
	if got != subcommand {
		t.Fatalf("selected subcommand = %#v, want %#v", got, subcommand)
	}
}
