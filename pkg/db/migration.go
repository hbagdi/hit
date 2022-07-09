package db

import (
	"context"
	"database/sql"
	"fmt"
)

func migrate(db *sql.DB) error {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE name='schema_migrations';`)
	if err != nil {
		return err
	}
	defer rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if !rows.Next() {
		err := initSchemaMigration(db)
		if err != nil {
			return err
		}
	}
	err = doMigrate(db, migrations)
	if err != nil {
		return err
	}
	return nil
}

func initSchemaMigration(sql *sql.DB) error {
	_, err := sql.Exec("create table schema_migrations(" +
		"id varchar primary key, count int)")
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %v", err)
	}
	_, err = sql.Exec(`insert into schema_migrations values('current_state',0);`)
	if err != nil {
		return fmt.Errorf("init schema_migrations: %v", err)
	}
	return nil
}

var migrations = []string{
	`create table if not exists hits(id integer primary key);`,
	`alter table hits add column hit_request_id text;`,
	`alter table hits add column created_at integer;`,
	`alter table hits add column http_request_method text;`,
	`alter table hits add column http_request_host text;`,
	`alter table hits add column http_request_path text;`,
	`alter table hits add column http_request_query_string text;`,
	`alter table hits add column http_request_headers text;`,
	`alter table hits add column http_request_body text;`,
	`alter table hits add column http_response_code integer;`,
	`alter table hits add column http_response_headers text;`,
	`alter table hits add column http_response_body text;`,
	`alter table hits add column http_response_status text;`,
}

func doMigrate(db *sql.DB, migrations []string) error {
	currentState, err := currentState(db)
	if err != nil {
		return err
	}
	if len(migrations) == currentState {
		return nil
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %v", err)
	}
	defer func() {
		// TODO(hbagdi): add logger
		_ = tx.Rollback()
	}()
	for i := currentState; i < len(migrations); i++ {
		_, err := tx.Exec(migrations[i])
		if err != nil {
			return fmt.Errorf("migration(%d): %v", i, err)
		}
	}
	err = updateCurrentState(tx, len(migrations))
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction: %v", err)
	}
	return nil
}

func updateCurrentState(tx *sql.Tx, newState int) error {
	_, err := tx.Exec(`update schema_migrations set count=? where id='current_state';`, newState)
	if err != nil {
		return fmt.Errorf("update current state: %v", err)
	}
	return nil
}

func currentState(db *sql.DB) (int, error) {
	rows, err := db.Query(`select count from schema_migrations where id='current_state';`)
	if err != nil {
		return 0, fmt.Errorf("read current state: %v", err)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("read current state rows: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, fmt.Errorf("no current_state in schema_migrations: possible" +
			" database corruption")
	}
	var currentState int
	err = rows.Scan(&currentState)
	if err != nil {
		return 0, fmt.Errorf("scan current state query: %v", err)
	}
	return currentState, nil
}
