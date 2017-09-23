package storage

const (
	STATUS_PENDING   = 1
	STATUS_ERROR     = 2
	STATUS_FAILED    = 3
	STATUS_COMPLETED = 4
)

type FileModel struct {
	Id         int
	Url        string
	Hash       string
	Resolution string
	BitRate    string
}

type LogModel struct {
	FileId  int
	Status  int
	Message string
}
