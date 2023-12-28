package main

import (
	"github.com/mattermost/mattermost-server/v6/model"
)

func main() {
	a := &app{logger: newLogger(), config: loadConfig()}
	a.client = model.NewAPIv4Client(a.config.Server.String())
	a.client.SetToken(a.config.Token)
	login(a)
	findTeam(a)
	setupShutdown(a)
	listenEvents(a)
}
