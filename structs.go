package main

import (
	"net/url"
	"os"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/rs/zerolog"

	_ "github.com/joho/godotenv/autoload"
)

// application struct to hold the dependencies for our bot
type application struct {
	config                    config
	logger                    zerolog.Logger
	mattermostClient          *model.Client4
	mattermostWebSocketClient *model.WebSocketClient
	mattermostUser            *model.User
	mattermostChannel         *model.Channel
	mattermostTeam            *model.Team
}

type config struct {
	mattermostUserName string
	mattermostTeamName string
	mattermostToken    string
	mattermostChannel  string
	mattermostServer   *url.URL
}

func loadConfig() config {
	var settings config

	settings.mattermostTeamName = os.Getenv("MM_TEAM")
	settings.mattermostUserName = os.Getenv("MM_USERNAME")
	settings.mattermostToken = os.Getenv("MM_TOKEN")
	settings.mattermostChannel = os.Getenv("MM_CHANNEL")
	settings.mattermostServer, _ = url.Parse(os.Getenv("MM_SERVER"))

	return settings
}
