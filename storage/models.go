package storage

type FileModel struct {
	Url        string
	Hash       string
	Resolution string
	BitRate    string
}

type LogModel struct {
	Url     string
	Status  int
	Message string
}
