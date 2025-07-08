package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/vetrovegor/kushfinds-backend/internal/app"
	"github.com/vetrovegor/kushfinds-backend/internal/config"
	"go.uber.org/zap"

	_ "github.com/vetrovegor/kushfinds-backend/docs"
)

//	@title			Kushfinds API
//	@version		1.0
//	@description	API Server for Kushfinds application

//	@host		localhost:8080
//	@BasePath	/api

// @securityDefinitions.apikey	ApiKeyAuth
// @in							header
// @name						Authorization
func main() {
	cfg := config.MustLoad()

	log, _ := zap.NewDevelopment()
	defer log.Sync()

	app := app.NewApp(log, *cfg)

	go func() {
		app.MustRun()
	}()

	log.Info("server started", zap.String("addr", cfg.Address))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	// app.Shutdown(context.Background())

	log.Info("server shutting down")
}
