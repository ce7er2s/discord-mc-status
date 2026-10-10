package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"

	"github.com/mcstatus-io/mcutil/v4/response"
	"github.com/mcstatus-io/mcutil/v4/status"
)

func getenv[T string | int](k string, def T) T {
	v, ok := os.LookupEnv(k)
	if !ok {
		return def
	}

	var out any = v
	if _, isInt := any(def).(int); isInt {
		n, err := strconv.Atoi(v)
		if err != nil {
			log.Fatalf("bad %s=%q: %v", k, v, err)
		}
		out = n
	}
	return out.(T)
}

var (
	mcAddress      = flag.String("address", getenv("MC_ADDR", "127.0.0.1"), "address to monitor")
	mcPort         = flag.Int("port", getenv("MC_PORT", 25565), "explicit port to connect to")
	removeCommands = flag.Bool("rmcmd", true, "remove registered slash commands after shutdown")
)

func main() {
	flag.Parse()

	var logFile io.Writer
	logFile, err := os.Open("current.log")
	if err != nil {
		logFile = io.Discard
	}

	var logLevel log.Level = log.DebugLevel
	if v, ok := os.LookupEnv("LOG_LEVEL"); ok {
		level, err := log.ParseLevel(v)
		if err != nil {
			logLevel = level
		}
	}

	l := log.NewWithOptions(io.MultiWriter(os.Stdout, logFile), log.Options{
		ReportTimestamp: true,
		ReportCaller:    true,
		Level:           logLevel,
	})

	discordToken, ok := os.LookupEnv("DISCORD_TOKEN")
	if !ok {
		l.Errorf("no discord token provided")
		os.Exit(1)
	}

	discord, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		l.Errorf("can't login discord bot: %v", err)
		os.Exit(1)
	}

	err = discord.Open()
	if err != nil {
		l.Errorf("can't login discord bot: %v", err)
		os.Exit(1)
	}
	defer discord.Close()

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "about",
			Description: fmt.Sprintf("Tracking server status at %s:%i", *mcAddress, *mcPort),
		},
	}

	commandHandlers := map[string]func(*discordgo.Session, *discordgo.InteractionCreate){
		"about": func(s *discordgo.Session, m *discordgo.InteractionCreate) {
			l.Debugf("dispatched to \"about\" handler")
			err := s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "aboutme message",
				},
			})

			if err != nil {
				l.Errorf("can't proceed interaction \"about\": %v", err)
			}
		},
	}

	var registeredCommands []*discordgo.ApplicationCommand
	for _, command := range commands {
		cmd, err := discord.ApplicationCommandCreate(discord.State.User.ID, "", command)
		if err != nil {
			l.Errorf("interaction \"%s\" registration failed: %v", command.Name)
		} else {
			registeredCommands = append(registeredCommands, cmd)
		}
	}

	discord.AddHandler(func(s *discordgo.Session, m *discordgo.InteractionCreate) {
		cmd := m.ApplicationCommandData().Name
		l.Debugf("got interaction \"%s\", dispatching...", cmd)
		if handler, ok := commandHandlers[cmd]; ok {
			handler(s, m)
		}
	})

	sc := make(chan os.Signal, 1)
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		cancel := context.CancelFunc(func() {})
		defer func() { cancel() }()

		for {
			select {
			case <-sc:
				return
			case t := <-ticker.C:
				ctx, cancel := context.WithDeadline(context.Background(), t.Add(5*time.Second))
				resp, err := fetchMcServerStatus(ctx, *mcAddress, int64(*mcPort))
				cancel()

				<-ctx.Done()

				var status string
				if err != nil {
					status = "Status: 🔴 offline"
				} else {
					status = createStatusString(resp)
				}

				err = discord.UpdateCustomStatus(status)
				if err != nil {
					l.Errorf("can't update status: %v", err)
				}
			}
		}
	}()

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	<-sc

	l.Info("bot stopped. cleaning up...")

	if *removeCommands {
		l.Info("removing commands...")

		for _, command := range registeredCommands {
			err := discord.ApplicationCommandDelete(discord.State.User.ID, "", command.ID)
			if err != nil {
				l.Errorf("cannot delete '%v' command: %v", command.Name, err)
			}
		}
	}
}

func fetchMcServerStatus(ctx context.Context, address string, port int64) (*response.StatusModern, error) {
	response, err := status.Modern(ctx, address, uint16(port))
	if err != nil {
		return nil, err
	}

	return response, err
}

func createStatusString(status *response.StatusModern) string {
	return fmt.Sprintf("Status: 🟢 online (%d / %d players)", *status.Players.Online, *status.Players.Max)
}
