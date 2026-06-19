package dgr

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type Dgr struct {
	Session             *discordgo.Session
	mu                  sync.RWMutex
	interactionHandlers map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate)
	buttonPool          map[string]ButtonInvoker
	commands            []*discordgo.ApplicationCommand
}

func (d *Dgr) initLocked() {
	if d.interactionHandlers == nil {
		d.interactionHandlers = make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate))
	}
	if d.buttonPool == nil {
		d.buttonPool = make(map[string]ButtonInvoker)
	}
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
	if d == nil || d.Session == nil {
		return errors.New("dgr: nil session")
	}
	if d.Session.State == nil || d.Session.State.User == nil {
		return errors.New("dgr: session user is not available; open the session before syncing commands")
	}

	d.mu.RLock()
	commands := append([]*discordgo.ApplicationCommand(nil), d.commands...)
	d.mu.RUnlock()

	_, err := d.Session.ApplicationCommandBulkOverwrite(
		d.Session.State.User.ID,
		guildID,
		commands,
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
	if d == nil || d.Session == nil {
		return nil
	}
	return d.Session.Close()
}

func (d *Dgr) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if d == nil || i == nil || i.Interaction == nil {
		return
	}

	switch i.Type {

	case discordgo.InteractionApplicationCommand:
		name := i.ApplicationCommandData().Name
		d.mu.RLock()
		handler := d.interactionHandlers[name]
		d.mu.RUnlock()
		if handler != nil {
			handler(s, i)
		}

	case discordgo.InteractionMessageComponent:
		customID := i.MessageComponentData().CustomID
		d.mu.RLock()
		btn := d.buttonPool[customID]
		d.mu.RUnlock()
		if btn != nil {
			btn.Invoke(s, i.Interaction)
		}

	default:
		return
	}
}
