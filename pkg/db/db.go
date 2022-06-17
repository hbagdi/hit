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

func NewStore(opts StoreOpts) (*Store, error) {
	db, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open db file: %v", err)
	}
	if opts.Logger == nil {
		return nil, fmt.Errorf("no logger")
	}
	return &Store{
		db:     db,
		logger: opts.Logger,
	}, nil
}
