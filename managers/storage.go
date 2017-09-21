package managers

import (
	"github.com/Sirupsen/logrus"
	"github.com/dk13danger/media-service/config"
	"github.com/dk13danger/media-service/storage"
)

const (
	STATUS_PENDING   = 1
	STATUS_ERROR     = 2
	STATUS_FAILED    = 3
	STATUS_COMPLETED = 4
)

type LogMessage struct {
	Status  int
	Message string
	FileUrl string
}

type FileMessage struct {
	Url        string
	Hash       string
	Resolution string
	BitRate    string
}

type StorageManager struct {
	logger    *logrus.Logger
	storage   storage.Storager
	logInput  chan *LogMessage
	fileInput chan *FileMessage
	done      chan struct{}
}

func NewStorageManager(
	logger *logrus.Logger,
	storage storage.Storager,
	cfg *config.StorageManager,
) *StorageManager {
	return &StorageManager{
		logger:    logger,
		storage:   storage,
		logInput:  make(chan *LogMessage, cfg.LogChannelSize),
		fileInput: make(chan *FileMessage, cfg.FileChannelSize),
		done:      make(chan struct{}, 1),
	}
}

func (sm *StorageManager) Run() (chan<- *LogMessage, chan<- *FileMessage) {
	go func() {
		for {
			select {
			case msg, ok := <-sm.logInput:
				if !ok {
					sm.logInput = nil
					break
				}
				id, err := sm.storage.GetFileIdByUrl(msg.FileUrl)
				if err != nil {
					sm.logger.Errorf("Error while fetching file id: %v", err)
					break
				}
				err = sm.storage.InsertLog(&storage.LogModel{
					FileId:  id,
					Status:  msg.Status,
					Message: msg.Message,
				})
				if err != nil {
					sm.logger.Errorf("Error while inserting logInput message: %v", err)
				}
			case file, ok := <-sm.fileInput:
				if !ok {
					sm.fileInput = nil
					break
				}
				err := sm.storage.InsertFile(&storage.FileModel{
					Url:        file.Url,
					Hash:       file.Hash,
					BitRate:    file.BitRate,
					Resolution: file.Resolution,
				})
				if err != nil {
					sm.logger.Errorf("Error while inserting file: %v", err)
				}
			}

			if sm.logInput == nil && sm.fileInput == nil {
				sm.done <- struct{}{}
				return
			}
		}
	}()

	return sm.logInput, sm.fileInput
}

// Stop stops storage process (writing to database, filesystem, whatever..).
// Queue channel is closed and all requests will be storaged
func (sm *StorageManager) Stop() {
	close(sm.logInput)
	close(sm.fileInput)
	sm.logger.Debug("Storage queues closed")
	<-sm.done
}
