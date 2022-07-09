package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/hbagdi/hit/pkg/model"
	"github.com/hbagdi/hit/pkg/util"
	_ "github.com/mattn/go-sqlite3" // sqlite driver
	"go.uber.org/zap"
)

func init() {
	util.EnsureCacheDirs()
}

type Store struct {
	db     *sql.DB
	logger *zap.Logger
}

const loadLatestQuery = `select 
hit_request_id,
created_at,
http_request_method,
http_request_host,
http_request_path,
http_request_query_string,
http_request_body,
http_response_code,
http_response_body
from hits
where hit_request_id=@hitRequestID
order by created_at desc limit 1;`

func (s *Store) LoadLatestHitForID(ctx context.Context, hitRequestID string) (model.Hit, error) {
	rows := s.db.QueryRowContext(ctx, loadLatestQuery,
		sql.Named("hitRequestID", hitRequestID),
	)
	if err := rows.Err(); err != nil {
		return model.Hit{}, err
	}
	var hit model.Hit
	err := rows.Scan(&hit.HitRequestID, &hit.CreatedAt, &hit.Request.Method,
		&hit.Request.Host, &hit.Request.Path, &hit.Request.QueryString,
		&hit.Request.Body, &hit.Response.Code, &hit.Response.Body)
	if err != nil {
		return model.Hit{}, err
	}
	return hit, nil
}

const saveQuery = `insert into hits(
hit_request_id,
created_at,
http_request_method,
http_request_host,
http_request_path,
http_request_query_string,
http_request_headers,
http_request_body,
http_response_code,
http_response_status,
http_response_headers,
http_response_body
)
values(
@hitRequestID,
@createdAt,
@httpRequestMethod,
@httpRequestHost,
@httpRequestPath,
@httpRequestQueryString,
@httpRequestHeaders,
@httpRequestBody,
@httpResponseCode,
@httpResponseStatus,
@httpResponseHeaders,
@httpResponseBody
);`

func (s *Store) Save(ctx context.Context, hit model.Hit) error {
	requestHeaders, err := json.Marshal(hit.Request.Header)
	if err != nil {
		return fmt.Errorf("marshal HTTP headers into json: %v", err)
	}
	responseHeaders, err := json.Marshal(hit.Response.Header)
	if err != nil {
		return fmt.Errorf("marshal HTTP headers into json: %v", err)
	}
	_, err = s.db.ExecContext(ctx, saveQuery,
		sql.Named("hitRequestID", hit.HitRequestID),
		sql.Named("createdAt", time.Now().Unix()),
		sql.Named("httpRequestMethod", hit.Request.Method),
		sql.Named("httpRequestHost", hit.Request.Host),
		sql.Named("httpRequestPath", hit.Request.Path),
		sql.Named("httpRequestQueryString", hit.Request.QueryString),
		sql.Named("httpRequestHeaders", string(requestHeaders)),
		sql.Named("httpRequestBody", hit.Request.Body),
		sql.Named("httpResponseCode", hit.Response.Code),
		sql.Named("httpResponseStatus", hit.Response.Status),
		sql.Named("httpResponseHeaders", string(responseHeaders)),
		sql.Named("httpResponseBody", hit.Response.Body),
	)
	if err != nil {
		return fmt.Errorf("execute sql: %v", err)
	}

	return nil
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
	cacheDir, err := util.HitCacheDir()
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
		return nil, fmt.Errorf("ensure migrations up to date: %v", err)
	}
	return &Store{
		db:     db,
		logger: opts.Logger,
	}, nil
}
