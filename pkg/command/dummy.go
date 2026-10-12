package command

import (
	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

type Dummy struct {
	l    *log.Logger
	name string
}

func newDummy(l *log.Logger, name string) *Dummy {
	return &Dummy{
		l:    l,
		name: name,
	}
}

func (d *Dummy) GetApplicationCommand() *discordgo.ApplicationCommand {
	return nil
}

func (d *Dummy) InteractionCreateHandler(client *discordgo.Session, i *discordgo.InteractionCreate) {
	d.l.Debugf("command \"%s\" dispatched to dummy: do nothing", d.name)
}
