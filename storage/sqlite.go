package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Sirupsen/logrus"
	_ "github.com/mattn/go-sqlite3"
)

type storage struct {
	logger                *logrus.Logger
	db                    *sql.DB
	insertFilesStmt       *sql.Stmt
	insertLogStmt         *sql.Stmt
	selectFilesStmt       *sql.Stmt
	selectFilesByUrlStmt  *sql.Stmt
	selectFileIdByUrlStmt *sql.Stmt
}

type Storager interface {
	GetFileIdByUrl(url string) (int, error)
	GetStatistic() ([]byte, error)
	GetStatisticByUrl(url string) ([]byte, error)
	InsertLog(model *LogModel) error
	InsertFile(model *FileModel) error
}

func NewSqliteStorage(logger *logrus.Logger, dbPath string) Storager {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		panic(fmt.Sprintf("Can't open db: %v", err))
	}
	s, err := prepareStatements(logger, db)
	if err != nil {
		panic(fmt.Sprintf("Can't prepare db statetement: %v", err))
	}
	return s
}

func (s *storage) GetFileIdByUrl(url string) (int, error) {
	rows, err := s.selectFileIdByUrlStmt.Query(url)
	if err != nil {
		return -1, err
	}
	defer rows.Close()

	var id int
	for rows.Next() {
		if err = rows.Scan(&id); err != nil {
			return -1, err
		}
	}

	return id, nil
}

func (s *storage) GetStatisticByUrl(url string) ([]byte, error) {
	return getStatistic(s.selectFilesByUrlStmt, url)
}

func (s *storage) GetStatistic() ([]byte, error) {
	return getStatistic(s.selectFilesStmt)
}

func (s *storage) InsertFile(model *FileModel) error {
	_, err := s.insertFilesStmt.Exec(model.Url, model.Hash, model.BitRate, model.Resolution)
	return err
}

func (s *storage) InsertLog(model *LogModel) error {
	_, err := s.insertLogStmt.Exec(model.FileId, model.Status, model.Message)
	return err
}

func prepareStatements(logger *logrus.Logger, db *sql.DB) (Storager, error) {
	insertFilesStmt, err := db.Prepare("INSERT INTO files(url, hash, resolution, bitrate) VALUES (?,?,?,?)")
	if err != nil {
		return nil, err
	}

	insertLogStmt, err := db.Prepare("INSERT INTO log(file_id, status, message) VALUES (?,?,?)")
	if err != nil {
		return nil, err
	}

	selectFilesStmt, err := db.Prepare(`
		SELECT f.id, f.url, f.hash, f.bitrate, f.resolution, l.status, l.message
		  FROM files f
		  JOIN log l
			ON l.file_id = f.id
	`)
	if err != nil {
		return nil, err
	}

	selectFilesByUrlStmt, err := db.Prepare(`
		SELECT f.id, f.url, f.hash, f.bitrate, f.resolution, l.status, l.message
		  FROM files f
		  JOIN log l
			ON l.file_id = f.id
		 WHERE f.url = ?
	`)
	if err != nil {
		return nil, err
	}

	selectFileIdByUrlStmt, err := db.Prepare(`SELECT id FROM files WHERE url = ?`)
	if err != nil {
		return nil, err
	}

	return &storage{
		logger:                logger,
		db:                    db,
		insertFilesStmt:       insertFilesStmt,
		insertLogStmt:         insertLogStmt,
		selectFilesStmt:       selectFilesStmt,
		selectFilesByUrlStmt:  selectFilesByUrlStmt,
		selectFileIdByUrlStmt: selectFileIdByUrlStmt,
	}, nil
}

func getStatistic(stmt *sql.Stmt, args ...interface{}) ([]byte, error) {
	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var id int
	var url, hash, bitrate, resolution, status, message string

	files := make(map[int]map[string]interface{})
	for rows.Next() {
		if err = rows.Scan(&id, &url, &hash, &bitrate, &resolution, &status, &message); err != nil {
			return nil, err
		}

		log := map[string]string{
			"status":  status,
			"message": message,
		}

		if file, ok := files[id]; ok {
			file["log"] = append(file["log"].([]map[string]string), log)
			continue
		}

		file := make(map[string]interface{}, 0)
		file["url"] = url
		file["hash"] = hash
		file["log"] = []map[string]string{log}

		files[id] = file
	}

	b, err := json.Marshal(files)
	if err != nil {
		return nil, err
	}

	return b, nil
}
