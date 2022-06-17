package db

import (
	"context"
	"database/sql"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func getDB(t *testing.T) *sql.DB {
	t.Helper()
	dirName, err := ioutil.TempDir("/tmp", "hit-dev-test-*")
	require.NoError(t, err)
	dsn := genDSN(filepath.Join(dirName, "test.db"))
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(dirName)
	})
	return db
}

func TestInitSchemaMigration(t *testing.T) {
	db := getDB(t)
	err := initSchemaMigration(db)
	require.NoError(t, err)

	res, err := db.Query("select * from schema_migrations;")
	require.NoError(t, err)
	require.NoError(t, res.Err())
	defer res.Close()
	require.True(t, res.Next(), "a single row exists")
	type row struct {
		id    string
		count int
	}
	var r row
	require.NoError(t, res.Scan(&r.id, &r.count))
	require.Equal(t, r.id, "current_state")
	require.Equal(t, r.count, 0)
}

func TestCurrentState(t *testing.T) {
	db := getDB(t)
	err := initSchemaMigration(db)
	require.NoError(t, err)

	state, err := currentState(db)
	require.NoError(t, err)
	require.Equal(t, 0, state)
}

func TestUpdateCurrentState(t *testing.T) {
	db := getDB(t)
	err := initSchemaMigration(db)
	require.NoError(t, err)

	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	err = updateCurrentState(tx, 2)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	state, err := currentState(db)
	require.NoError(t, err)
	require.Equal(t, 2, state)
}

func TestDoMigrate(t *testing.T) {
	db := getDB(t)
	err := initSchemaMigration(db)
	require.NoError(t, err)

	require.NoError(t, doMigrate(db, []string{
		`create table foo(id text primary key);`,
		`create table bar(id text primary key);`,
	}))

	state, err := currentState(db)
	require.NoError(t, err)
	require.Equal(t, 2, state)
}

func TestDoMigrateWithRealMigrations(t *testing.T) {
	db := getDB(t)
	err := initSchemaMigration(db)
	require.NoError(t, err)

	require.NoError(t, doMigrate(db, migrations))

	state, err := currentState(db)
	require.NoError(t, err)
	require.Equal(t, len(migrations), state)
}
