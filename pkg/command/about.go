package command

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/ce7er2s/discord-mc-status/pkg/config"

	clog "github.com/charmbracelet/log"
)

type aboutCommand struct {
	l   *clog.Logger
	cfg *config.Config

	// unused
	cmd *discordgo.ApplicationCommand
}

func newAboutCommandWithDummy(l *clog.Logger, cfg *config.Config, client *discordgo.Session) Command {
	cmd, err := newAboutCommand(l, cfg, client)

	if err != nil {
		l.Errorf("can't create about command (falling back to dummy): %v", err)
		return newDummy(l, "about")
	}

	return cmd
}

func newAboutCommand(l *clog.Logger, cfg *config.Config, client *discordgo.Session) (*aboutCommand, error) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "about",
		Description: "Minecraft server details",
	}

	cmd, err := client.ApplicationCommandCreate(client.State.User.ID, "", cmd)

	return &aboutCommand{
		l:   l,
		cfg: cfg,
		cmd: cmd,
	}, err
}

func (cmd *aboutCommand) GetApplicationCommand() *discordgo.ApplicationCommand {
	return cmd.cmd
}

func (cmd *aboutCommand) InteractionCreateHandler(client *discordgo.Session, i *discordgo.InteractionCreate) {
	cmd.l.Debugf("dispatched to \"about\" handler")
	err := client.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Tracking server status at %s:%d", cmd.cfg.Address, cmd.cfg.Port),
		},
	})

	if err != nil {
		cmd.l.Errorf("can't proceed interaction \"about\": %v", err)
	}
}
