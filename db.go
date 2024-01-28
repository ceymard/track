package main

import (
	"database/sql"
	"log"
	"sync"

	_ "embed"

	_ "github.com/mattn/go-sqlite3"
)

/*
example de query au jour le jour.

select
  *
from
(select
  date(ts / 1000, 'unixepoch') as date,
  time(sum(dur / 1000), 'unixepoch') as time,
  case when name regexp '^\d+$|__i3_scratch' then '__scratch' else name end as name
from track
--where name not regexp '^\d+$'
group by
	date(ts / 1000, 'unixepoch'),
	case when name regexp '^\d+$|__i3_scratch' then '__scratch' else name end
) order by date, time desc
*/

var lock sync.RWMutex

func DBrw(fn func(db *sql.DB) error) error {
	lock.Lock()
	defer lock.Unlock()
	return fn(db)
}

func DBro(fn func(db *sql.DB) error) error {
	lock.RLock()
	defer lock.RUnlock()
	return fn(db)
}

var db *sql.DB
var stmt *sql.Stmt

func DBNotify(name string, ts int64, dur int64) error {
	return DBrw(func(db *sql.DB) error {
		_, err := stmt.Exec(name, ts, dur)
		return err
	})
}

//go:embed db_init.sql
var SQL_INIT string

func OpenDB() error {
	var err error
	db, err = sql.Open("sqlite3", "/home/cey/track.db?_journal_mode=wal")
	if err != nil {
		log.Fatal(err)
	}

	if _, err = db.Exec(SQL_INIT); err != nil {
		log.Fatal(err)
		return err
	}

	if stmt, err = db.Prepare(`INSERT INTO track(name, ts, dur) values (?, ?, ?)`); err != nil {
		return err
	}

	return nil
}
