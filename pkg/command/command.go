package command

import (
	"github.com/bwmarrin/discordgo"
	"github.com/ce7er2s/discord-mc-status/pkg/config"
	clog "github.com/charmbracelet/log"
)

type Dispatcher struct {
	l   *clog.Logger
	cfg *config.Config

	client   *discordgo.Session
	commands map[string]Command
}

type Command interface {
	InteractionCreateHandler(*discordgo.Session, *discordgo.InteractionCreate)
	GetApplicationCommand() *discordgo.ApplicationCommand
}

func NewDispatcher(l *clog.Logger, cfg *config.Config, client *discordgo.Session) Dispatcher {
	commands := map[string]Command{
		"about": newAboutCommandWithDummy(l, cfg, client),
	}

	return Dispatcher{
		l:   l,
		cfg: cfg,

		client:   client,
		commands: commands,
	}
}

func (d Dispatcher) CreateCommandHandler() func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		name := i.ApplicationCommandData().Name
		d.l.Debugf("got interaction \"%s\", dispatching...", name)
		if command, ok := d.commands[name]; ok {
			command.InteractionCreateHandler(s, i)
		} else {
			d.l.Errorf("command \"%s\" is not configured (strange)", name)
		}
	}
}

func (d Dispatcher) Delete() {
	for name, command := range d.commands {
		d.l.Debugf("removing command \"%s\"...", name)
		appCmd := command.GetApplicationCommand()
		if appCmd != nil {
			err := d.client.ApplicationCommandDelete(d.client.State.User.ID, "", appCmd.ID)
			if err != nil {
				d.l.Errorf("cannot delete '%v' command: %v", name, err)
			}
		}
	}
}
