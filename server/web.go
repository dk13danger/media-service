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
	storage storage.Storager
	logger  *logrus.Logger
	cfg     *config.Server
}

func NewServer(
	storage storage.Storager,
	logger *logrus.Logger,
	cfg *config.Server,
) *Server {
	return &Server{
		storage: storage,
		logger:  logger,
		cfg:     cfg,
	}
}

func (s *Server) Run(downloadQueue chan<- *service.Task) {
	files, err := s.storage.Select1InterruptFiles()
	if err != nil {
		s.logger.Errorf("Can't get list of interrupt tasks: %v", err)
		return
	}

	for _, f := range files {
		s.logger.Infof("Continue downloading interrupted tasks (count: %d)..", len(files))
		downloadQueue <- &service.Task{
			Url:  f.Url,
			Hash: f.Hash,
		}
	}

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
}
