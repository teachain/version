package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/teachain/version/config"
	"github.com/teachain/version/internal/handler"
	"github.com/teachain/version/internal/model"
	"github.com/teachain/version/internal/poller"
	"github.com/teachain/version/internal/repository"
	"github.com/teachain/version/internal/service"
	gh "github.com/teachain/version/pkg/github"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to YAML config file")
	flag.StringVar(&configPath, "c", "", "path to YAML config file (shorthand)")
	flag.Parse()

	if configPath == "" {
		configPath = os.Getenv("CONFIG_FILE")
	}
	if configPath == "" {
		configPath = "./config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	logger, err := newLogger(cfg.LogLevel)
	if err != nil {
		log.Fatalf("logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	eng, err := repository.NewEngine(cfg.DBDSN)
	if err != nil {
		logger.Fatal("engine", zap.Error(err))
	}
	if err := eng.Sync(new(model.Application), new(model.Version)); err != nil {
		logger.Fatal("sync schema", zap.Error(err))
	}

	appRepo := repository.NewApplicationXormRepository(eng)
	verRepo := repository.NewVersionXormRepository(eng)
	releaseRepo := gh.NewClient(cfg.GitHubToken)
	appSvc := service.NewApplicationService(appRepo)
	verSvc := service.NewVersionService(verRepo)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	handler.NewApplicationHandler(r, appSvc)
	handler.NewVersionHandler(r, verSvc)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	p2 := poller.New(appRepo, verRepo, releaseRepo, cfg.PollInterval, cfg.PollConcurrency, logger)
	go p2.Run(ctx)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen", zap.Error(err))
		}
	}()

	logger.Info("server started", zap.Int("port", cfg.HTTPPort))
	<-ctx.Done()
	logger.Info("shutting down")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		logger.Error("shutdown", zap.Error(err))
	}
	os.Exit(0)
}

func newLogger(level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	switch level {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	return cfg.Build()
}
