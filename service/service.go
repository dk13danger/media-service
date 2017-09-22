package service

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/dk13danger/media-service/config"
	"github.com/dk13danger/media-service/managers"
)

type Config struct {
	Dir         string
	Attempts    int
	Workers     int
	ChannelSize int
}

type Service struct {
	logger         *logrus.Logger
	cacheManager   *managers.CacheManager
	storageManager *managers.StorageManager
	cfg            *config.Service
	regexp         *regexp.Regexp
	wg             *sync.WaitGroup
	inputTasks     chan Task
	outputLog      chan<- *managers.LogMessage
	outputFile     chan<- *managers.FileMessage
	done           chan struct{}
}

type Task struct {
	Url string
	MD5 string
}

func NewService(
	logger *logrus.Logger,
	cacheManager *managers.CacheManager,
	storageManager *managers.StorageManager,
	cfg *config.Service,
) *Service {
	if _, err := os.Stat(cfg.OutputDir); os.IsNotExist(err) {
		logger.Debugf("Service dir %q not exists yet. Trying to create", cfg.OutputDir)
		os.Mkdir(cfg.OutputDir, os.ModeDir)
	}
	return &Service{
		logger:         logger,
		cacheManager:   cacheManager,
		storageManager: storageManager,
		cfg:            cfg,
		regexp:         regexp.MustCompile(`(?m)^width=(\d+)\r*\n*height=(\d+)\r*\n*bit_rate=(\d+).*$`),
		wg:             &sync.WaitGroup{},
		inputTasks:     make(chan Task, cfg.ChannelSize),
		done:           make(chan struct{}, 1),
	}
}

func (s *Service) Run() chan<- Task {
	s.logger.Info("Starting storage manager")
	s.outputLog, s.outputFile = s.storageManager.Run()

	s.wg.Add(s.cfg.Workers)
	for i := 0; i < s.cfg.Workers; i++ {
		s.logger.Debugf("Starting download worker number: #%d", i)
		go func() {
			for t := range s.inputTasks {
				if file, err := s.processTask(t, 1); err != nil {
					s.logToStorage(managers.STATUS_FAILED, fmt.Sprintf("Error processing task: %v", err), t.Url)
				} else {
					if file != nil {
						s.logToStorage(managers.STATUS_COMPLETED, "File has been processed successfully!", t.Url)
						s.outputFile <- file
					}
				}
			}
			s.logger.Debugf("Stop download worker")
			s.wg.Done()
		}()
	}
	return s.inputTasks
}

func (s *Service) Stop() {
	close(s.inputTasks)
	s.logger.Debug("Service task queue closed")
	s.logger.Debugf("Wait while %d service workers stopping..", s.cfg.Workers)
	s.wg.Wait()

	s.logger.Debug("Stopping storage manager")
	s.storageManager.Stop()
	s.logger.Debug("Storage manager stopped")
}

func (s *Service) processTask(e Task, attempt int) (*managers.FileMessage, error) {
	if s.cacheManager.Get(e.Url) {
		msg := "File already downloading. Please wait.."
		s.logger.Debug(msg)
		s.outputLog <- &managers.LogMessage{managers.STATUS_PENDING, msg, e.Url}
		return nil, nil
	}

	s.cacheManager.Set(e.Url)

	if attempt > s.cfg.Attempts {
		return nil, fmt.Errorf("all attempts are spent (count: %s)", s.cfg.Attempts)
	}

	s.logger.Debugf("Processing service task. Attempt number: #%d", attempt)

	filePath, err := s.downloadFromUrl(e.Url)
	if err != nil {
		s.logToStorage(managers.STATUS_ERROR, fmt.Sprintf("Error while downloading file: %v", err), e.Url)
		return s.processTask(e, attempt+1)
	}

	if err := s.validateChecksum(filePath, e.MD5); err != nil {
		s.logToStorage(managers.STATUS_ERROR, fmt.Sprintf("Error while validating checksum: %v", err), e.Url)
		return s.processTask(e, attempt+1)
	}

	bitRate, resolution, err := s.getMediaInfo(filePath)
	if err != nil {
		return nil, err
	}

	s.cacheManager.Remove(e.Url)

	return &managers.FileMessage{
		Url:        e.Url,
		Hash:       e.MD5,
		BitRate:    bitRate,
		Resolution: resolution,
	}, nil
}

func (s *Service) downloadFromUrl(url string) (string, error) {
	tokens := strings.Split(url, "/")
	filePath := fmt.Sprintf("%s/%s", s.cfg.OutputDir, tokens[len(tokens)-1])

	if _, err := os.Stat(filePath); os.IsExist(err) {
		s.logger.Infof("File %q already exists on local filesystem. Skip downloading", filePath)
		return filePath, nil
	}

	output, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("error while creating file %q: %v", filePath, err)
	}
	defer output.Close()

	s.logToStorage(
		managers.STATUS_PENDING,
		fmt.Sprintf("Downloading from url: %q to file: %q started..", url, filePath),
		url,
	)
	start := time.Now()

	response, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("error while downloading url %q: %v", url, err)
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		return "", fmt.Errorf("error while copying to file %q: %v", filePath, err)
	}

	s.logToStorage(
		managers.STATUS_PENDING,
		fmt.Sprintf("Done! Time elapsed: %q (%d bytes downloaded)", time.Since(start), n),
		url,
	)

	return filePath, nil
}

func (s *Service) validateChecksum(filePath, checksum string) error {
	s.logger.Debugf("Get MD5 hash from file: %q", filePath)
	hash, err := s.getMD5(filePath)
	if err != nil {
		return fmt.Errorf("error while getting md5 hash: %v", err)
	}
	if hash != checksum {
		return fmt.Errorf("suspusious file. Checksum from url: %q mismatch with file hash: %q", checksum, hash)
	}
	return nil
}

func (s *Service) getMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	hashInBytes := hash.Sum(nil)[:16]

	return hex.EncodeToString(hashInBytes), nil
}

func (s *Service) getMediaInfo(filePath string) (bitRate, resolution string, err error) {
	cmdName := "ffprobe"
	cmdArgs := []string{
		"-v", "error", "-show_entries", "stream=width,height,bit_rate", "-of", "default=noprint_wrappers=1", filePath,
	}

	s.logger.Debugf("Get media info from file: %q by shell command: %q", filePath, cmdName)
	cmdOut, err := exec.Command(cmdName, cmdArgs...).Output()
	if err != nil {
		return "", "", fmt.Errorf("there was an error running %q command: %v", cmdName, err)
	}

	match := s.regexp.FindStringSubmatch(string(cmdOut))

	return fmt.Sprintf("%sx%s", match[1], match[2]), string(match[3]), nil
}

func (s *Service) logToStorage(status int, msg, url string) {
	switch status {
	case managers.STATUS_PENDING, managers.STATUS_COMPLETED:
		s.logger.Info(msg)
	case managers.STATUS_FAILED, managers.STATUS_ERROR:
		s.logger.Error(msg)
	}
	s.outputLog <- &managers.LogMessage{status, msg, url}
}
