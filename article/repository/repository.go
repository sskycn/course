// Copyright 2020-2024 The GoStartKit Authors. All rights reserved.
// Use of this source code is governed by a AGPL
// license that can be found in the LICENSE file.
// NOTE: This file should not be edited
// see https://gostartkit.com/docs/code for more information.
package repository

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"app.gostartkit.com/go/article/config"
	"app.gostartkit.com/go/article/helper"
	"pkg.gostartkit.com/utils"
)

const (
	_pageSize    = 10
	_maxPageSize = 1000
)

var (
	_writeDatabase *sql.DB
	_readDatabases []*sql.DB
	_once          sync.Once
)

// Init config
func Init(cfg *config.DatabaseCluster) error {

	var err error

	_once.Do(func() {

		_writeDatabase, err = openWriteDatabase(cfg)

		if err == nil {
			_readDatabases, err = openReadDatabases(cfg)
		}
	})

	return err
}

// Close databases
func Close() error {

	var err error

	if _writeDatabase != nil {

		if err1 := _writeDatabase.Close(); err1 != nil {
			err = err1
		}
	}

	for _, r := range _readDatabases {

		if r != nil {

			if err1 := r.Close(); err1 != nil {
				err = err1
			}
		}
	}

	return err
}

// openWriteDatabase of config.database.write
func openWriteDatabase(cfg *config.DatabaseCluster) (*sql.DB, error) {
	var (
		username string
		password string
	)

	if cfg.Write.Username == "" || cfg.Write.Password == "" {
		username = cfg.Username
		password = cfg.Password
	} else {
		username = cfg.Write.Username
		password = cfg.Write.Password
	}

	db, err := sql.Open(cfg.Driver, fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true",
		username,
		password,
		cfg.Write.Host,
		cfg.Write.Port,
		cfg.Database,
		cfg.Charset))

	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(0)

	return db, nil
}

// openReadDatabases of config.database.read
func openReadDatabases(cfg *config.DatabaseCluster) ([]*sql.DB, error) {

	var (
		readDatabases []*sql.DB
		username      string
		password      string
	)

	for _, r := range *cfg.Read {

		if r.Username == "" || r.Password == "" {
			username = cfg.Username
			password = cfg.Password
		} else {
			username = r.Username
			password = r.Password
		}

		db, err := sql.Open(cfg.Driver, fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true",
			username,
			password,
			r.Host,
			r.Port,
			cfg.Database,
			cfg.Charset))

		if err != nil {
			return nil, err
		}

		db.SetMaxIdleConns(0)

		readDatabases = append(readDatabases, db)
	}

	return readDatabases, nil
}

func selectDB(databaseCluster []*sql.DB) *sql.DB {
	return databaseCluster[helper.RandMax(len(databaseCluster))]
}

func query(query string, args ...any) (*sql.Rows, error) {

	rows, err := selectDB(_readDatabases).Query(query, args...)

	if err != nil {
		if config.App().AppDebug {
			log.Printf("query: %s\n", query)
		}
		return nil, err
	}

	return rows, nil
}

func queryRow(query string, args ...any) *sql.Row {
	return selectDB(_readDatabases).QueryRow(query, args...)
}

func prepare(query string) (*sql.Stmt, error) {

	stmt, err := _writeDatabase.Prepare(query)

	if err != nil {
		if config.App().AppDebug {
			log.Printf("prepare: %s\n", query)
		}
		return nil, err
	}

	return stmt, nil
}

func txPrepare(tx *sql.Tx, query string) (*sql.Stmt, error) {

	stmt, err := tx.Prepare(query)

	if err != nil {
		if config.App().AppDebug {
			log.Printf("txPrepare: %s\n", query)
		}
		return nil, err
	}

	return stmt, nil
}

func stmtExec(stmt *sql.Stmt, args ...any) (sql.Result, error) {

	result, err := stmt.Exec(args...)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func exec(query string, args ...any) (sql.Result, error) {

	result, err := _writeDatabase.Exec(query, args...)

	if err != nil {
		if config.App().AppDebug {
			log.Printf("exec: %s\n", query)
			log.Printf("args: %v\n", args)
		}
		return nil, err
	}

	return result, nil
}

func begin() (*sql.Tx, error) {
	return _writeDatabase.Begin()
}

func txExec(tx *sql.Tx, query string, args ...any) (sql.Result, error) {

	result, err := tx.Exec(query, args...)

	if err != nil {
		if config.App().AppDebug {
			log.Printf("txExec: %s\n", query)
			log.Printf("args: %v\n", args)
		}
		return nil, err
	}

	return result, nil
}

func rollback(tx *sql.Tx) error {
	return tx.Rollback()
}

func commit(tx *sql.Tx) error {
	return tx.Commit()
}

func now() *time.Time {
	return utils.Now()
}

// max get max key value
func max(tableName string, key string, appID uint64, appNum uint64) (uint64, error) {

	sqlx := "SELECT MAX(`" + key + "`) FROM `" + tableName + "` WHERE `" + key + "` % ? = ? "

	row := queryRow(sqlx, appNum, appID)

	var val *uint64

	err := row.Scan(&val)

	if err != nil {
		return 0, err
	}

	if val == nil {
		return 0, nil
	}

	return *val, nil
}
