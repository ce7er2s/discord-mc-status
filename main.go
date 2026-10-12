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
	"github.com/ce7er2s/discord-mc-status/pkg/command"
	"github.com/ce7er2s/discord-mc-status/pkg/config"
	"github.com/ce7er2s/discord-mc-status/pkg/status"

	clog "github.com/charmbracelet/log"
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
			clog.Fatalf("bad %s=%q: %v", k, v, err)
		}
		out = n
	}
	return out.(T)
}

// TODO: normal flags

var (
	mcAddress             = flag.String("address", getenv("MC_ADDR", "127.0.0.1"), "address to monitor")
	mcPort                = flag.Int("port", getenv("MC_PORT", 25565), "explicit port to connect to")
	removeCommands        = flag.Bool("rm", true, "remove registered slash commands after shutdown")
	statusUpdateInterval  = flag.Duration("update-interval", time.Second*10, "online status update interval")
	serverTimeoutInterval = flag.Duration("server-timeout", time.Second*5, "server response timeout")

	l   *clog.Logger
	cfg *config.Config
)

func init() {

	flag.Parse()

	var logFile io.Writer
	logFile, err := os.Open("current.log")
	if err != nil {
		logFile = io.Discard
	}

	var logLevel clog.Level = clog.DebugLevel
	if v, ok := os.LookupEnv("LOG_LEVEL"); ok {
		level, err := clog.ParseLevel(v)
		if err != nil {
			logLevel = level
		}
	}

	l = clog.NewWithOptions(io.MultiWriter(os.Stdout, logFile), clog.Options{
		ReportTimestamp: true,
		ReportCaller:    true,
		Level:           logLevel,
	})

	cfg = &config.Config{
		Address:         *mcAddress,
		Port:            *mcPort,
		UpdateInterval:  *statusUpdateInterval,
		TimeoutInterval: *serverTimeoutInterval,
	}
}

func main() {

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

	dispatcher := command.NewDispatcher(l, cfg, discord)

	discord.AddHandler(dispatcher.CreateCommandHandler())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go status.UpdateOnlineStatus(ctx, l, cfg, discord)

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")

	<-ctx.Done()

	l.Info("bot stopped. cleaning up...")

	if *removeCommands {
		dispatcher.Delete()
	}
}
