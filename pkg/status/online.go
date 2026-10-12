package status

import (
	"context"
	"strings"
	"text/template"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ce7er2s/discord-mc-status/pkg/config"
	clog "github.com/charmbracelet/log"
	"github.com/mcstatus-io/mcutil/v4/response"
	"github.com/mcstatus-io/mcutil/v4/status"
)

var (
	onlineStatusTemplate, _  = template.New("online").Parse("Status: 🟢 online ({{ .Players.Online }} / {{ .Players.Max }} players)")
	offlineStatusTemplate, _ = template.New("offline").Parse("Status: 🔴 offline")
)

func UpdateOnlineStatus(ctx context.Context, l *clog.Logger, cfg *config.Config, client *discordgo.Session) {
	ticker := time.NewTicker(cfg.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			resp, err := fetchMcServerStatus(ctx, cfg)

			var statusBuilder strings.Builder
			if err != nil {
				err = offlineStatusTemplate.Execute(&statusBuilder, resp)
			} else {
				err = onlineStatusTemplate.Execute(&statusBuilder, resp)
			}

			if err != nil {
				l.Errorf("got template error while updating status: %v", err)
				continue
			}

			err = client.UpdateCustomStatus(statusBuilder.String())
			if err != nil {
				l.Errorf("can't update status: %v", err)
			}
		}
	}
}

func fetchMcServerStatus(ctx context.Context, cfg *config.Config) (*response.StatusModern, error) {
	ctx, cancel := context.WithTimeout(ctx, cfg.TimeoutInterval)
	defer cancel()

	response, err := status.Modern(ctx, cfg.Address, uint16(cfg.Port))
	if err != nil {
		return nil, err
	}

	return response, err
}
