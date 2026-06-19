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
