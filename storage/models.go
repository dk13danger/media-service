package storage

type FileModel struct {
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
