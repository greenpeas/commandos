package app

import (
	"commandos/internal/config"
	mainRepo "commandos/internal/repository/main"
	"commandos/internal/services/logger"
	"commandos/internal/types"
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"
)

type repo interface {
	AddCommand(imei string, name string, params string, author string) (int32, string, error)
	ListCommands(imei string, offset int32, limit int32, newOnly bool) ([]types.Command, error)
	UpdCommand(id int32, responseDate time.Time, response, rawRequest, rawResponse string) (int32, int64, error)
	UpdTry(id int32) error
}

type App struct {
	cfg    config.Config
	ctx    context.Context
	logger *logger.Logger
	repo   repo
}

func NewApp(cfg config.Config, ctx context.Context, logger *logger.Logger, repo repo) *App {

	return &App{
		cfg,
		ctx,
		logger,
		repo,
	}
}

func InitAndRun(configPath string) {

	log.Println("Init application")

	log.Println(time.Now().Zone())

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	_ = cancel

	cfg := config.Init(configPath)
	logger := logger.NewLogger()

	if !cfg.EnableService {
		log.Println("Service is disabled by config")
		return
	}

	dbpool, connErr := connectPg(cfg.Database.Psql.Url, 3, ctx) // 3 попытки

	if connErr != nil {
		log.Println(connErr)
		return
	}

	defer dbpool.Close()

	repo := mainRepo.NewSpoRepo(dbpool, logger, ctx)

	a := NewApp(cfg, ctx, logger, repo)
	a.Run()

}
