package main

import (
	"net/url"
	"os"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/rs/zerolog"

	_ "github.com/joho/godotenv/autoload"
)

type app struct {
	config   config
	logger   zerolog.Logger
	client   *model.Client4
	wsClient *model.WebSocketClient
	user     *model.User
	team     *model.Team
}

type config struct {
	TeamName string
	Token    string
	Server   *url.URL
}

func loadConfig() config {
	var settings config
	settings.TeamName = os.Getenv("MM_TEAM")
	settings.Token = os.Getenv("MM_TOKEN")
	settings.Server, _ = url.Parse(os.Getenv("MM_SERVER"))
	return settings
}

func newLogger() zerolog.Logger {
	return zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC822}).With().Timestamp().Logger()
}
