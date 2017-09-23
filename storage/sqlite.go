package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Sirupsen/logrus"
	_ "github.com/mattn/go-sqlite3"
)

type storage struct {
	logger                   *logrus.Logger
	db                       *sql.DB
	insertFileStmt           *sql.Stmt
	insertLogStmt            *sql.Stmt
	selectFilesStmt          *sql.Stmt
	selectFilesByUrlStmt     *sql.Stmt
	selectFileStmt           *sql.Stmt
	selectInterruptFilesStmt *sql.Stmt
	updateFileStmt           *sql.Stmt
	checkFileIsCompletedStmt           *sql.Stmt
}

type Storager interface {
	CheckFileIsCompleted(fileId int) (bool, error)
	GetStatistic() ([]byte, error)
	GetStatisticByUrl(url, hash string) ([]byte, error)
	InsertLog(model *LogModel) (int, error)
	InsertFile(model *FileModel) (int, error)
	SelectFile(url, hash string) (int, error)
	Select1InterruptFiles() ([]FileModel, error)
	UpdateFile(model *FileModel) (int, error)
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

func (s *storage) CheckFileIsCompleted(fileId int) (bool, error) {
	rows, err := s.checkFileIsCompletedStmt.Query(fileId)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	var id int
	for rows.Next() {
		if err = rows.Scan(&id); err == nil {
			return true, err
		}
	}
	return false, nil
}

func (s *storage) SelectFile(url, hash string) (int, error) {
	rows, err := s.selectFileStmt.Query(url, hash)
	if err != nil {
		return -1, err
	}
	defer rows.Close()

	var id int
	for rows.Next() {
		if err = rows.Scan(&id); err != nil {
			return -1, err
		} else {
			return id, nil
		}
	}
	return -1, nil
}

func (s *storage) Select1InterruptFiles() ([]FileModel, error) {
	rows, err := s.selectInterruptFilesStmt.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var id int
	var url, hash string
	ret := make([]FileModel, 0)

	for rows.Next() {
		if err = rows.Scan(&id, &url, &hash); err != nil {
			return nil, err
		}
		ret = append(ret, FileModel{
			Id:   id,
			Url:  url,
			Hash: hash,
		})
	}
	return ret, nil
}

func (s *storage) GetStatisticByUrl(url, hash string) ([]byte, error) {
	return getStatistic(s.selectFilesByUrlStmt, url, hash)
}

func (s *storage) GetStatistic() ([]byte, error) {
	return getStatistic(s.selectFilesStmt)
}

func (s *storage) InsertFile(model *FileModel) (int, error) {
	res, err := s.insertFileStmt.Exec(model.Url, model.Hash, model.BitRate, model.Resolution)
	if err != nil {
		return -1, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return -1, err
	}
	return int(id), err
}

func (s *storage) UpdateFile(model *FileModel) (int, error) {
	_, err := s.updateFileStmt.Exec(model.BitRate, model.Resolution, model.Id)
	if err != nil {
		return -1, err
	}
	return model.Id, err
}

func (s *storage) InsertLog(model *LogModel) (int, error) {
	_, err := s.insertLogStmt.Exec(model.FileId, model.Status, model.Message)
	return -1, err
}

func getStatistic(stmt *sql.Stmt, args ...interface{}) ([]byte, error) {
	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var id, url, hash, bitrate, resolution, status, message string

	files := make(map[string]map[string]interface{})
	for rows.Next() {
		if err = rows.Scan(&id, &url, &hash, &bitrate, &resolution, &status, &message); err != nil {
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

func prepareStatements(logger *logrus.Logger, db *sql.DB) (Storager, error) {
	insertFileStmt, err := db.Prepare("INSERT INTO files(url, hash, resolution, bitrate) VALUES (?,?,?,?)")
	if err != nil {
		return nil, err
	}

	insertLogStmt, err := db.Prepare("INSERT INTO log(file_id, status, message) VALUES (?,?,?)")
	if err != nil {
		return nil, err
	}

	selectFileStmt, err := db.Prepare("SELECT id FROM files WHERE url=? AND hash=?")
	if err != nil {
		return nil, err
	}

	selectInterruptFilesStmt, err := db.Prepare(fmt.Sprintf(`
		SELECT DISTINCT f.id, f.url, f.hash
	      FROM files f
	      LEFT JOIN log l
	        ON l.file_id = f.id
	     WHERE l.status NOT IN (%d, %d)
	        OR l.id IS NULL
	`, STATUS_COMPLETED, STATUS_FAILED))
	if err != nil {
		return nil, err
	}

	checkFileIsCompletedStmt, err := db.Prepare(fmt.Sprintf(`
		SELECT DISTINCT f.id
	      FROM files f
	      JOIN log l
	        ON l.file_id = f.id
	     WHERE l.status IN (%d)
	       AND f.id = ?
	`, STATUS_COMPLETED))
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
		   AND f.hash = ?
	`)
	if err != nil {
		return nil, err
	}

	updateFileStmt, err := db.Prepare(`UPDATE files SET bitrate=?, resolution=? WHERE id=?`)
	if err != nil {
		return nil, err
	}

	return &storage{
		logger:                   logger,
		db:                       db,
		insertFileStmt:           insertFileStmt,
		insertLogStmt:            insertLogStmt,
		selectFilesStmt:          selectFilesStmt,
		selectFilesByUrlStmt:     selectFilesByUrlStmt,
		selectFileStmt:           selectFileStmt,
		selectInterruptFilesStmt: selectInterruptFilesStmt,
		updateFileStmt:           updateFileStmt,
		checkFileIsCompletedStmt:           checkFileIsCompletedStmt,
	}, nil
}
