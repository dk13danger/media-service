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
	"github.com/dk13danger/media-service/storage"
)

type Service struct {
	logger       *logrus.Logger
	cacheManager *CacheManager
	storage      storage.Storager
	cfg          *config.Service
	regexp       *regexp.Regexp
	wg           *sync.WaitGroup
	inputTasks   chan *Task
	done         chan struct{}
}

type Task struct {
	Url  string
	Hash string
}

func NewService(
	storage storage.Storager,
	cacheManager *CacheManager,
	logger *logrus.Logger,
	cfg *config.Service,
) *Service {
	if _, err := os.Stat(cfg.OutputDir); os.IsNotExist(err) {
		logger.Debugf("Service dir %q not exists yet. Trying to create", cfg.OutputDir)
		os.Mkdir(cfg.OutputDir, os.ModeDir)
	}
	return &Service{
		logger:       logger,
		cacheManager: cacheManager,
		storage:      storage,
		cfg:          cfg,
		regexp:       regexp.MustCompile(`(?m)^width=(\d+)\r*\n*height=(\d+)\r*\n*bit_rate=(\d+).*$`),
		wg:           &sync.WaitGroup{},
		inputTasks:   make(chan *Task, cfg.ChannelSize),
		done:         make(chan struct{}, 1),
	}
}

func (s *Service) Run() chan<- *Task {
	s.wg.Add(s.cfg.Workers)
	for i := 0; i < s.cfg.Workers; i++ {
		s.logger.Debugf("Starting download worker number: #%d", i)
		go func() {
			for t := range s.inputTasks {
				key := fmt.Sprintf("%s-%s", t.Url, t.Hash)
				if s.cacheManager.Get(key) {
					s.logger.Debug("File already downloading. Please wait..")
					continue
				}

				if err := s.processTask(t, 1, key); err != nil {
					s.logger.Errorf("Error while processing task: %v", err)
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
}

func (s *Service) processTask(t *Task, attempt int, key string) error {
	s.cacheManager.Set(key)
	defer s.cacheManager.Remove(key)

	fileId, err := s.storage.SelectFile(t.Url, t.Hash)
	if err != nil {
		return fmt.Errorf("error while selecting file, url: %q, hash: %q", t.Url, t.Hash)
	}
	if fileId < 0 {
		fileId, err = s.storage.InsertFile(&storage.FileModel{
			Url:  t.Url,
			Hash: t.Hash,
		})
		if err != nil {
			return fmt.Errorf("error while inserting file, url: %q, hash: %q", t.Url, t.Hash)
		}
	}

	if attempt > s.cfg.Attempts {
		msg := fmt.Sprintf("all attempts are spent (count: %d)", s.cfg.Attempts)
		s.logToStorage(fileId, storage.STATUS_FAILED, msg)
		return fmt.Errorf(msg)
	}

	completed, err := s.storage.CheckFileIsCompleted(1)
	if err != nil {
		return fmt.Errorf("error while checking file: %v", err)
	}
	if completed {
		s.logger.Info("File already processed successfully. Skip.")
		return nil
	}

	s.logger.Debugf("Processing service task. Attempt number: #%d", attempt)
	s.logToStorage(fileId, storage.STATUS_PENDING, "Start processing task")

	filePath, err := s.download(fileId, t)
	if err != nil {
		s.logToStorage(fileId, storage.STATUS_ERROR, fmt.Sprintf("Error while downloading file: %v", err))
		return s.processTask(t, attempt+1, key)
	}

	if err := s.validateChecksum(filePath, t.Hash); err != nil {
		s.logToStorage(fileId, storage.STATUS_ERROR, fmt.Sprintf("Error while validating checksum: %v", err))
		return s.processTask(t, attempt+1, key)
	}

	bitRate, resolution, err := s.getMediaInfo(filePath)
	if err != nil {
		s.logToStorage(fileId, storage.STATUS_FAILED, fmt.Sprintf("Error while getting media info: %v", err))
		return fmt.Errorf("error while getting media info: %v", err)
	}

	_, err = s.storage.UpdateFile(&storage.FileModel{
		Id:         fileId,
		Url:        t.Url,
		Hash:       t.Hash,
		BitRate:    bitRate,
		Resolution: resolution,
	})
	if err != nil {
		return fmt.Errorf("error while updating file: %v", err)
	}
	s.logToStorage(fileId, storage.STATUS_COMPLETED, "Task completed")

	return nil
}

func (s *Service) download(fileId int, t *Task) (string, error) {
	tokens := strings.Split(t.Url, "/")
	filePath := fmt.Sprintf("%s/%s-%s", s.cfg.OutputDir, tokens[len(tokens)-1], t.Hash)

	if _, err := os.Stat(filePath); err == nil {
		if err = os.Remove(filePath); err != nil {
			return "", fmt.Errorf("can't remove file %q from local filesystem", filePath)
		}
	}

	output, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("error while creating file %q: %v", filePath, err)
	}
	defer output.Close()

	s.logToStorage(fileId, storage.STATUS_PENDING, fmt.Sprintf("Start downloading from url: %q", t.Url))
	start := time.Now()

	response, err := http.Get(t.Url)
	if err != nil {
		return "", fmt.Errorf("error while downloading url %q: %v", t.Url, err)
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		return "", fmt.Errorf("error while copying to file %q: %v", filePath, err)
	}

	s.logToStorage(fileId, storage.STATUS_PENDING, fmt.Sprintf(
		"Finish downloading. Time elapsed: %q (%d bytes downloaded), file path: %q",
		time.Since(start),
		n,
		filePath,
	))

	return filePath, nil
}

func (s *Service) validateChecksum(filePath, checksum string) error {
	s.logger.Debugf("Get hash from file: %q", filePath)
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

func (s *Service) logToStorage(fileId, status int, msg string) error {
	switch status {
	case storage.STATUS_PENDING, storage.STATUS_COMPLETED:
		s.logger.Info(msg)
	case storage.STATUS_FAILED, storage.STATUS_ERROR:
		s.logger.Error(msg)
	}

	_, err := s.storage.InsertLog(&storage.LogModel{
		FileId:  fileId,
		Message: msg,
		Status:  status,
	})
	if err != nil {
		return err
	}

	return nil
}
