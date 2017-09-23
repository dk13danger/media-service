package storage

const (
	STATUS_PENDING   = 2
	STATUS_ERROR     = 4
	STATUS_FAILED    = 5
	STATUS_COMPLETED = 3
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
