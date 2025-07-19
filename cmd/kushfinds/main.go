package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vetrovegor/kushfinds-backend/internal/app"
	"github.com/vetrovegor/kushfinds-backend/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

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

	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	log, _ := config.Build()

	app := app.New(log, *cfg)

	go func() {
		app.MustRun()
	}()

	log.Info("server started", zap.String("addr", cfg.Address))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	app.Shutdown(ctx)

	log.Info("server shutting down")
}
