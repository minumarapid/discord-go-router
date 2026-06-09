package dgr

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type Dgr struct {
	Session             *discordgo.Session
	interactionHandlers map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate)
	buttonPool          map[string]ButtonInvoker
	commands            []*discordgo.ApplicationCommand
}

func New(token string) (*Dgr, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	d := &Dgr{
		Session:             session,
		interactionHandlers: make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate)),
		buttonPool:          make(map[string]ButtonInvoker),
	}

	session.AddHandler(d.onInteraction)

	return d, nil
}

func (d *Dgr) SyncCommands(guildID string) error {
	_, err := d.Session.ApplicationCommandBulkOverwrite(
		d.Session.State.User.ID,
		guildID,
		d.commands,
	)
	return err
}

func (d *Dgr) Run(guildID string) error {
	err := d.Session.Open()
	if err != nil {
		return err
	}
	defer d.Session.Close()

	if err := d.SyncCommands(guildID); err != nil {
		log.Printf("Failed to sync commands: %v", err)
	}
	log.Println("Bot started successfully. Press Ctrl+C to stop.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("Shutting down the bot...")
	return nil
}

func (d *Dgr) Stop() error {
	return d.Session.Close()
}

func (d *Dgr) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {

	case discordgo.InteractionApplicationCommand:
		name := i.ApplicationCommandData().Name
		if handler, ok := d.interactionHandlers[name]; ok {
			handler(s, i)
		}

	case discordgo.InteractionMessageComponent:
		customID := i.MessageComponentData().CustomID
		if btn, ok := d.buttonPool[customID]; ok {
			btn.Invoke(s, i.Interaction)
		}

	default:
		return
	}
}
