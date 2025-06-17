package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/NailUsmanov/gophermart/internal/app"
	"github.com/NailUsmanov/gophermart/internal/storage"
	"github.com/NailUsmanov/gophermart/pkg/config"
	"go.uber.org/zap"
)

func main() {

	// Cоздаём предустановленный регистратор zap
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// Создаем регистратор SugaredLogger
	sugar := logger.Sugar()

	// Создаем канал для сигналов
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Контекст, который отменится при сигнале
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Отдельная горутина ждёт сигнал и вызывает cancel()
	go func() {
		<-sigChan
		cancel()
	}()

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dbStorage, err := storage.NewDataBaseStorage(cfg.DataBaseURI)
	if err != nil {
		sugar.Fatalf("failed to connect to database: %v", err)
	}

	applictaion := app.NewApp(dbStorage, sugar, cfg.Accural)
	if err := applictaion.Run(ctx, cfg.RunAddr); err != nil {
		sugar.Fatalln(err)
	}

}
