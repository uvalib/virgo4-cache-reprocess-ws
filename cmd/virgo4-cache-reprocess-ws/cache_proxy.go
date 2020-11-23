package main

import (
	"fmt"
	"log"
	"time"

	dbx "github.com/go-ozzo/ozzo-dbx"
	_ "github.com/lib/pq"
)

// ErrNotInCache - item not in the cache
var ErrNotInCache = fmt.Errorf("item not in cache")

// returned when a record is not located
var notFound = "sql: no rows in result set"

// log a warning if any postgres request takes longer than this
var getRequestTimeLimit = int64(100)

// CacheProxy - our interface
type CacheProxy interface {
	Get(string) (*CacheRecord, error)
	Healthcheck() bool
}

// our implementation
type cacheProxyImpl struct {
	tableName string
	db        *dbx.DB
}

type CacheRecord struct {
	ID      string `db:"id"`
	Type    string `db:"type"`
	Source  string `db:"source"`
	Payload string `db:"payload"`
}

//
// NewCacheProxy - our factory
//
func NewCacheProxy(config *ServiceConfig) (CacheProxy, error) {

	impl := &cacheProxyImpl{}

	connStr := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d connect_timeout=%d sslmode=disable",
		config.PostgresUser, config.PostgresPass, config.PostgresDatabase, config.PostgresHost, config.PostgresPort, 30)

	db, err := dbx.MustOpen("postgres", connStr)
	if err != nil {
		log.Printf("ERROR: connecting to database: %s", err.Error())
		return nil, err
	}

	// uncomment for SQL logging
	//db.LogFunc = log.Printf

	impl.tableName = config.PostgresTable
	impl.db = db
	return impl, nil
}

//
// get the specified items from the cache
//
func (ci *cacheProxyImpl) Get(key string) (*CacheRecord, error) {

	start := time.Now()
	var search CacheRecord
	err := ci.db.Select().From(ci.tableName).Where(dbx.HashExp{"id": key}).One(&search)
	elapsed := int64(time.Since(start) / time.Millisecond)
	ci.warnIfSlow(elapsed, getRequestTimeLimit, fmt.Sprintf("CacheGet (id:%s)", key))

	if err != nil {
		if err.Error() == notFound {
			return nil, ErrNotInCache
		}
		return nil, err
	}

	return &search, nil
}

func (ci *cacheProxyImpl) Healthcheck() bool {
	var result struct {
		Result int `db:"result"`
	}

	q := ci.db.NewQuery("select 1 as result")
	err := q.One(&result)
	if err != nil {
		log.Printf("WARNING: healthcheck failure: %s", err.Error())
		return false
	}

	return true
}

// sometimes it is interesting to know if our DB queries are slow
func (ci *cacheProxyImpl) warnIfSlow(elapsed int64, limit int64, prefix string) {

	if elapsed > limit {
		log.Printf("INFO: %s elapsed %d ms", prefix, elapsed)
	}
}

//
// end of file
//
