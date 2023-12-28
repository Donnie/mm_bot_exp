package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/rs/zerolog"
)

func main() {

	app := &application{
		logger: zerolog.New(
			zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: time.RFC822,
			},
		).With().Timestamp().Logger(),
	}

	app.config = loadConfig()
	app.logger.Info().Str("config", fmt.Sprint(app.config)).Msg("")

	setupGracefulShutdown(app)

	// Create a new mattermost client.
	app.mattermostClient = model.NewAPIv4Client(app.config.mattermostServer.String())

	// Login.
	app.mattermostClient.SetToken(app.config.mattermostToken)

	if user, resp, err := app.mattermostClient.GetUser("me", ""); err != nil {
		app.logger.Fatal().Err(err).Msg("Could not log in")
	} else {
		app.logger.Debug().Interface("user", user).Interface("resp", resp).Msg("")
		app.logger.Info().Msg("Logged in to mattermost")
		app.mattermostUser = user
	}

	// Find and save the bot's team to app struct.
	if team, resp, err := app.mattermostClient.GetTeamByName(app.config.mattermostTeamName, ""); err != nil {
		app.logger.Fatal().Err(err).Msg("Could not find team. Is this bot a member ?")
	} else {
		app.logger.Debug().Interface("team", team).Interface("resp", resp).Msg("")
		app.mattermostTeam = team
	}

	// Listen to live events coming in via websocket.
	listenToEvents(app)
}

func setupGracefulShutdown(app *application) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			if app.mattermostWebSocketClient != nil {
				app.logger.Info().Msg("Closing websocket connection")
				app.mattermostWebSocketClient.Close()
			}
			app.logger.Info().Msg("Shutting down")
			os.Exit(0)
		}
	}()
}

func listenToEvents(app *application) {
	var err error
	failCount := 0
	for {
		app.mattermostWebSocketClient, err = model.NewWebSocketClient4(
			fmt.Sprintf("wss://%s", app.config.mattermostServer.Host+app.config.mattermostServer.Path),
			app.mattermostClient.AuthToken,
		)
		if err != nil {
			app.logger.Warn().Err(err).Msg("Mattermost websocket disconnected, retrying")
			failCount += 1
			// TODO: backoff based on failCount and sleep for a while.
			continue
		}
		app.logger.Info().Msg("Mattermost websocket connected")

		app.mattermostWebSocketClient.Listen()

		for event := range app.mattermostWebSocketClient.EventChannel {
			// Launch new goroutine for handling the actual event.
			// If required, you can limit the number of events beng processed at a time.
			go handleWebSocketEvent(app, event)
		}
	}
}

func handleWebSocketEvent(app *application, event *model.WebSocketEvent) {
	// Only process 'Posted' events.
	if event.EventType() != model.WebsocketEventPosted {
		return
	}

	// Since this event is a post, unmarshal it to (*model.Post)
	post := &model.Post{}
	err := json.Unmarshal([]byte(event.GetData()["post"].(string)), &post)
	if err != nil {
		app.logger.Error().Err(err).Msg("Could not cast event to *model.Post")
		return
	}

	// Ignore messages sent by this bot itself.
	if post.UserId == app.mattermostUser.Id {
		return
	}

	// Get the channel information
	channel, _, appErr := app.mattermostClient.GetChannel(post.ChannelId, "")
	if appErr != nil {
		app.logger.Error().Err(appErr).Msg("Failed to get channel")
		return
	}

	// Check if the channel is a direct message channel
	if channel.Type != "D" {
		return
	}

	// Handle the direct message post.
	handleDirectMessagePost(app, post)
}

func handleDirectMessagePost(app *application, post *model.Post) {
	// Log the received message for debugging purposes.
	app.logger.Debug().Str("message", post.Message).Msg("Received direct message")

	// Reply with "You said: [message text]"
	replyText := fmt.Sprintf("You said: %s", post.Message)
	sendMsgToDirectChannel(app, replyText, post.ChannelId)
}

func sendMsgToDirectChannel(app *application, msg string, channelId string) {
	// Create a new post in the specified channel.
	post := &model.Post{
		ChannelId: channelId,
		Message:   msg,
	}

	// Send the message.
	if _, _, err := app.mattermostClient.CreatePost(post); err != nil {
		app.logger.Error().Err(err).Str("ChannelID", channelId).Msg("Failed to send direct message")
	}
}
