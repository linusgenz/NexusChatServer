package config

import (
	"database/sql"
)

type DatabaseConfig struct {
	Driver   string
	Source   string
	MaxConns int
}

// DatabasePool represents a simple connection pool.
type DatabasePool struct {
	DB *sql.DB
}

func NewDatabasePool(dbConfig DatabaseConfig) (*DatabasePool, error) {
	db, err := sql.Open(dbConfig.Driver, dbConfig.Source)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(dbConfig.MaxConns)

	return &DatabasePool{
		DB: db,
	}, nil
}

func (pool *DatabasePool) RollbackOrCommit(tx *sql.Tx, success bool) error {
	if !success {
		if err := tx.Rollback(); err != nil {
			return err
		}
	} else {
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

var dbPool *DatabasePool

func InitDatabase(pool *DatabasePool) {
	dbPool = pool
}

func UseDBPool() *DatabasePool {
	return dbPool
}
