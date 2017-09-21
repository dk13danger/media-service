package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/dk13danger/media-service/config"
	"github.com/dk13danger/media-service/service"
	"github.com/dk13danger/media-service/storage"
	"github.com/gin-gonic/gin"
)

type Server struct {
	service *service.Service
	storage storage.Storager
	logger  *logrus.Logger
	cfg     *config.Server
}

func NewServer(
	service *service.Service,
	storage storage.Storager,
	logger *logrus.Logger,
	cfg *config.Server,
) *Server {
	return &Server{
		service: service,
		storage: storage,
		logger:  logger,
		cfg:     cfg,
	}
}

func (s *Server) Run() {
	s.logger.Debug("Starting service")
	downloadQueue := s.service.Run()

	router := gin.Default()
	router.GET("/dl", downloadHandler(downloadQueue, s.logger))
	router.GET("/st", statisticHandler(s.storage, s.logger))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.Port),
		Handler: router,
	}

	go func() {
		s.logger.Info("Starting server")
		srv.ListenAndServe()
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout.
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	s.logger.Println("Server shutdown started..")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cfg.ShutdownTimeout)*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		s.logger.Fatalf("Server shutdown err: %v", err)
	}

	s.logger.Println("Server shutdown finished")
	s.Stop()
}

func (s *Server) Stop() {
	s.logger.Debug("Stopping service")
	s.service.Stop()
	s.logger.Debug("Service stopped")
}
