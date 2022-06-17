package db

import (
	"database/sql"
	"fmt"
	"path/filepath"

	cachePkg "github.com/hbagdi/hit/pkg/cache"
	_ "github.com/mattn/go-sqlite3" // sqlite driver
	"go.uber.org/zap"
)

type Store struct {
	db     *sql.DB
	logger *zap.Logger
}

func (s *Store) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("close database: %v", err)
	}
	return nil
}

type StoreOpts struct {
	Logger *zap.Logger
}

var dbFilePath string

func init() {
	const dbFilename = "hit-requests.db"
	cacheDir, err := cachePkg.HitCacheDir()
	if err != nil {
		panic(fmt.Sprintf("failed to find cache dir: %v", err))
	}
	dbFilePath = filepath.Join(cacheDir, dbFilename)
}

func genDSN(fileName string) string {
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=500", fileName)
	return dsn
}

func NewStore(opts StoreOpts) (*Store, error) {
	dsn := genDSN(dbFilePath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db file: %v", err)
	}
	if opts.Logger == nil {
		return nil, fmt.Errorf("no logger")
	}
	err = migrate(db)
	if err != nil {
		return nil, err
	}
	return &Store{
		db:     db,
		logger: opts.Logger,
	}, nil
}
