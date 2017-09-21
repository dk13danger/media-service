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
	storageManager := managers.NewStorageManager(logger, sqLiteProvider, &cfg.StorageManager)
	cacheManager := managers.NewCacheManager(logger, &cfg.CacheManager)

	srv := service.NewService(logger, cacheManager, storageManager, &cfg.Service)

	web := server.NewServer(srv, sqLiteProvider, logger, &cfg.Server)
	web.Run()
}
