package main

import (
	"context"
	"fmt"

	"github.com/paper-trade-chatbot/be-common/cache"
	"github.com/paper-trade-chatbot/be-common/database"
	"github.com/paper-trade-chatbot/be-match/cronjob"
	"github.com/paper-trade-chatbot/be-match/service"

	"github.com/paper-trade-chatbot/be-common/config"
	"github.com/paper-trade-chatbot/be-common/global"
	"github.com/paper-trade-chatbot/be-common/logging"
	"github.com/paper-trade-chatbot/be-common/server"
	"github.com/paper-trade-chatbot/be-match/pubsub"
)

func main() {

	global.Alive = true

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logging.Initialize(ctx)
	defer logging.Finalize()

	cache.Initialize(ctx)
	defer cache.Finalize()

	database.Initialize(ctx)
	defer database.Finalize()

	service.Initialize(ctx)
	defer service.Finalize(ctx)

	pubsub.Initialize(ctx)
	defer pubsub.Finalize(ctx)

	initConfig()

	address := fmt.Sprintf("%s:%s",
		config.GetString("SERVER_LISTEN_ADDRESS"),
		config.GetString("SERVER_LISTEN_PORT"))
	httpServer := server.CreateHttpServer(ctx, address)

	go cronjob.Cron()

	global.Ready = true

	logging.Info(ctx, "Initialization complete, listening on %s...", address)
	if err := httpServer.ListenAndServe(); err != nil {
		logging.Info(ctx, err.Error())
	}

}

func initConfig() {
	global.Initialize()

}
