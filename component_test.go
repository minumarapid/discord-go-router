package dgr

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestButtonRowToComponent(t *testing.T) {
	type yesArgs struct{}
	type noArgs struct{}

	row := &ButtonRow{
		Buttons: []ButtonComponent{
			&Button[yesArgs]{Text: "Yes", CustomID: "yes", Style: discordgo.SuccessButton},
			&Button[noArgs]{Text: "No", CustomID: "no", Style: discordgo.DangerButton},
		},
	}

	component := row.ToComponent()
	actionRow, ok := component.(discordgo.ActionsRow)
	if !ok {
		t.Fatalf("component is %T, want discordgo.ActionsRow", component)
	}
	if len(actionRow.Components) != 2 {
		t.Fatalf("got %d components, want 2", len(actionRow.Components))
	}

	first, ok := actionRow.Components[0].(discordgo.Button)
	if !ok {
		t.Fatalf("first component is %T, want discordgo.Button", actionRow.Components[0])
	}
	if first.Label != "Yes" || first.CustomID != "yes" || first.Style != discordgo.SuccessButton {
		t.Fatalf("unexpected first button: %#v", first)
	}

	second, ok := actionRow.Components[1].(discordgo.Button)
	if !ok {
		t.Fatalf("second component is %T, want discordgo.Button", actionRow.Components[1])
	}
	if second.Label != "No" || second.CustomID != "no" || second.Style != discordgo.DangerButton {
		t.Fatalf("unexpected second button: %#v", second)
	}
}

func TestNewButtonRowRejectsMoreThanFiveButtons(t *testing.T) {
	ctx := &Context[struct{}]{}

	_, err := ctx.NewButtonRow(
		&Button[struct{}]{},
		&Button[struct{}]{},
		&Button[struct{}]{},
		&Button[struct{}]{},
		&Button[struct{}]{},
		&Button[struct{}]{},
	)
	if err == nil {
		t.Fatal("expected error for more than 5 buttons")
	}
}
