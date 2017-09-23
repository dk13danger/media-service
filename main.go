package main

import (
	"flag"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/dk13danger/media-service/config"
	"github.com/dk13danger/media-service/managers"
	"github.com/dk13danger/media-service/server"
	"github.com/dk13danger/media-service/service"
	"github.com/dk13danger/media-service/storage"
	"github.com/gin-gonic/gin"
)

var cfgFile = flag.String("config", "cfg/dev.yml", "path to config (default: cfg/dev.yml)")

func main() {
	flag.Parse()
	cfg := config.MustInit(*cfgFile)

	logger := logrus.New()
	if os.Getenv("DEBUG_MODE") == "true" {
		logger.Level = logrus.DebugLevel
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	sqLiteProvider := storage.NewSqliteStorage(logger, cfg.DbFilepath)
	cacheManager := managers.NewCacheManager(logger, &cfg.CacheManager)


	//if f, err := sqLiteProvider.CheckFileIsCompleted(1); err != nil {
	//	logger.Error(err)
	//} else {
	//	logger.Warn(f)
	//}
	//logger.Fatal()









	srv := service.NewService(sqLiteProvider, cacheManager, logger, &cfg.Service)
	downloadQueue := srv.Run()

	web := server.NewServer(sqLiteProvider, logger, &cfg.Server)
	web.Run(downloadQueue)

	srv.Stop()
	logger.Debug("Service stopped")
}
