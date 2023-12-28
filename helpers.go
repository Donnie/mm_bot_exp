package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"

	"github.com/mattermost/mattermost-server/v6/model"
)

func login(a *app) {
	user, _, err := a.client.GetUser("me", "")
	if err != nil {
		a.logger.Fatal().Err(err).Msg("Login failed")
	}
	a.logger.Info().Msg("Logged in")
	a.user = user
}

func findTeam(a *app) {
	team, _, err := a.client.GetTeamByName(a.config.TeamName, "")
	if err != nil {
		a.logger.Fatal().Err(err).Msg("Team not found")
	}
	a.logger.Info().Msg("Team Joined")
	a.team = team
}

func setupShutdown(a *app) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		if a.wsClient != nil {
			a.logger.Info().Msg("Closing WS connection")
			a.wsClient.Close()
		}
		a.logger.Info().Msg("Shutting down")
		os.Exit(0)
	}()
}

func listenEvents(a *app) {
	var err error
	for {
		a.wsClient, err = model.NewWebSocketClient4("wss://"+a.config.Server.Host+a.config.Server.Path, a.client.AuthToken)
		if err != nil {
			a.logger.Warn().Err(err).Msg("WS disconnected, retrying")
			continue
		}
		a.logger.Info().Msg("WS connected")
		a.wsClient.Listen()

		for event := range a.wsClient.EventChannel {
			go handleEvent(a, event)
		}
	}
}

func handleEvent(a *app, event *model.WebSocketEvent) {
	if event.EventType() != model.WebsocketEventPosted {
		return
	}

	post := &model.Post{}
	if err := json.Unmarshal([]byte(event.GetData()["post"].(string)), &post); err != nil {
		a.logger.Error().Err(err).Msg("Failed to unmarshal post")
		return
	}
	if post.UserId == a.user.Id {
		return
	}

	channel, _, err := a.client.GetChannel(post.ChannelId, "")
	if err != nil || channel.Type != "D" {
		return
	}

	handleDM(a, post)
}

func handleDM(a *app, post *model.Post) {
	a.logger.Info().Msg("DM received")
	sendMsg(a, fmt.Sprintf("You said: %s", post.Message), post.ChannelId)
}

func sendMsg(a *app, msg, channelId string) {
	post := &model.Post{ChannelId: channelId, Message: msg}
	if _, _, err := a.client.CreatePost(post); err != nil {
		a.logger.Error().Err(err).Str("ChannelID", channelId).Msg("Failed to send DM")
	}
}
