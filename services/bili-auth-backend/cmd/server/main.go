package main

import (
	"bili-auth-backend/internal/handler"
	"bili-auth-backend/internal/httpclient"
	"bili-auth-backend/internal/model"
	"bili-auth-backend/internal/service/auth"
	"bili-auth-backend/internal/store"
	"bili-auth-backend/internal/utils"
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg := model.LoadConfig()
	logger := utils.NewJSONLogger()

	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	st := store.NewMemorySessionStore(cfg.CleanupInterval, logger)
	st.StartCleanup()
	defer st.StopCleanup()

	client := httpclient.NewBilibiliAuthClient(
		cfg.HTTPTimeout,
		cfg.UserAgent,
		cfg.Referer,
		cfg.BiliGenerateURL,
		cfg.BiliPollURL,
		cfg.PollMaxRetries,
		cfg.PollRetryInterval,
	)
	authSvc := auth.NewService(st, client, cfg, logger)
	h := handler.NewAuthHandler(authSvc, cfg.Debug)
	liveClient := httpclient.NewBilibiliLiveClient(cfg.HTTPTimeout, cfg.UserAgent, cfg.Referer)
	liveHandler := handler.NewLiveHandler(authSvc, liveClient)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogMiddleware(logger))
	h.Register(r)
	liveHandler.Register(r)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server stopped", "error", err)
			os.Exit(1)
		}
	}()

	waitSignalAndShutdown(srv, logger)
}

func requestLogMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("http_request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds())
	}
}

func waitSignalAndShutdown(srv *http.Server, logger *slog.Logger) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
}
