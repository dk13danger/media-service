package server

import (
	"fmt"
	"net/http"
	net_url "net/url"

	"github.com/Sirupsen/logrus"
	"github.com/dk13danger/media-service/service"
	"github.com/dk13danger/media-service/storage"
	"github.com/gin-gonic/gin"
)

func downloadHandler(downloadQueue chan<- service.Task, logger *logrus.Logger) func(c *gin.Context) {
	return func(c *gin.Context) {
		url := c.Query("url")
		md5 := c.Query("md5")

		if err := validateQueryParams(url, md5); err != nil {
			msg := fmt.Sprintf("Bad request: %v", err)
			logger.Errorf(msg)
			c.JSON(http.StatusBadRequest, gin.H{"error": msg})
			return
		}

		downloadQueue <- service.Task{
			Url: url,
			MD5: md5,
		}
	}
}

func statisticHandler(storageProvider storage.Storager, logger *logrus.Logger) func(c *gin.Context) {
	return func(c *gin.Context) {
		url := c.Query("url")

		if url == "" {
			logger.Infof("Trying to get full statistics")
			b, err := storageProvider.GetStatistic()
			if err != nil {
				msg := fmt.Sprintf("Ooops: %v", err)
				logger.Errorf(msg)
				c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
				return
			}
			c.String(http.StatusOK, "%s\n", b)
			return
		}

		if _, err := net_url.ParseRequestURI(url); err != nil {
			msg := fmt.Sprintf("Bad request: %v", err)
			logger.Errorf(msg)
			c.JSON(http.StatusBadRequest, gin.H{"error": msg})
			return
		}

		logger.Infof("Trying to get statistics by url: %q", url)
		b, err := storageProvider.GetStatisticByUrl(url)
		if err != nil {
			msg := fmt.Sprintf("Ooops: %v", err)
			logger.Errorf(msg)
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		c.String(http.StatusOK, "%s\n", b)
	}
}

func validateQueryParams(url string, md5 string) error {
	if _, err := net_url.ParseRequestURI(url); err != nil {
		return err
	}
	if md5 == "" && len(md5) != 32 {
		return fmt.Errorf("MD5 length invalid. Must be: %d", 32)
	}
	return nil
}
