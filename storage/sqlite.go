package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Sirupsen/logrus"
	_ "github.com/mattn/go-sqlite3"
)

type storage struct {
	logger               *logrus.Logger
	db                   *sql.DB
	insertFilesStmt      *sql.Stmt
	insertLogStmt        *sql.Stmt
	selectFilesStmt      *sql.Stmt
	selectFilesByUrlStmt *sql.Stmt
}

type Storager interface {
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
	_, err := s.insertLogStmt.Exec(model.Url, model.Status, model.Message)
	return err
}

func prepareStatements(logger *logrus.Logger, db *sql.DB) (Storager, error) {
	insertFilesStmt, err := db.Prepare("INSERT INTO files(url, hash, resolution, bitrate) VALUES (?,?,?,?)")
	if err != nil {
		return nil, err
	}

	insertLogStmt, err := db.Prepare("INSERT INTO log(url, status, message) VALUES (?,?,?)")
	if err != nil {
		return nil, err
	}

	selectFilesStmt, err := db.Prepare(`
		SELECT f.url, f.hash, f.bitrate, f.resolution, l.status, l.message
		  FROM files f
		  JOIN log l
			ON l.url = f.url
	`)
	if err != nil {
		return nil, err
	}

	selectFilesByUrlStmt, err := db.Prepare(`
		SELECT f.url, f.hash, f.bitrate, f.resolution, l.status, l.message
		  FROM files f
		  JOIN log l
			ON l.url = f.url
		 WHERE f.url = ?
	`)
	if err != nil {
		return nil, err
	}

	return &storage{
		logger:               logger,
		db:                   db,
		insertFilesStmt:      insertFilesStmt,
		insertLogStmt:        insertLogStmt,
		selectFilesStmt:      selectFilesStmt,
		selectFilesByUrlStmt: selectFilesByUrlStmt,
	}, nil
}

func getStatistic(stmt *sql.Stmt, args ...interface{}) ([]byte, error) {
	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var url, hash, bitrate, resolution, status, message string

	files := make(map[string]map[string]interface{})
	for rows.Next() {
		if err = rows.Scan(&url, &hash, &bitrate, &resolution, &status, &message); err != nil {
			return nil, err
		}

		log := map[string]string{
			"status":  status,
			"message": message,
		}

		if file, ok := files[url]; ok {
			file["log"] = append(file["log"].([]map[string]string), log)
			continue
		}

		file := make(map[string]interface{}, 0)
		file["hash"] = hash
		file["bitrate"] = bitrate
		file["resolution"] = resolution
		file["log"] = []map[string]string{log}

		files[url] = file
	}

	b, err := json.Marshal(files)
	if err != nil {
		return nil, err
	}

	return b, nil
}
