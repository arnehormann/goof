package dbfetch

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
)

type querror struct {
	query string
	err   error
}

func (e querror) Error() string {
	return fmt.Sprintf("%v for query %q", e.err, e.query)
}

type fetcher struct {
	db    *sql.DB
	query string
	// use prepared statement; relevant for MySQL binary instead of text protocol
	asStmt bool
	// rows.Scan target pointers. Will be derived if nil
	dst []any
	// query arguments
	args []any
	// initCols is called before the first call to rows.Scan followed by yield;
	// it can still change dst.
	initCols func([]*sql.ColumnType, error) error
	// yield is called once per row
	yield func() error
}

func Fetch(db *sql.DB, query string) *fetcher {
	f := &fetcher{
		db:    db,
		query: query,
	}
	return f
}

func (f *fetcher) deriveScan() func([]*sql.ColumnType, error) error {
	// add a default function to derive scan types
	return func(cts []*sql.ColumnType, err error) error {
		if err != nil {
			return err
		}
		scan := make([]any, len(cts))
		for i, ct := range cts {
			v := reflect.New(ct.ScanType())
			scan[i] = v.Interface()
		}
		f.dst = scan
		return nil
	}
}

// UseStmt defines whether the query should be run as a prepared statement.
func (f *fetcher) UseStmt(p bool) *fetcher {
	f.asStmt = p
	return f
}

// ScanInto sets scan destinations.
// It expects a slice of pointers to all variables for the column values.
//
// Use ScanInto and Yield to easily access column data each row:
//
//		var k string, v int
//		accessCountPrev24h := make(map[string]int)
//		err := dbfetch.Fetch(db, `select login, count(*) from accesses group by login where ts > now() - interval 24 hour`).
//	     	Prepared(true).
//		    ScanInto(&k, &v).
//		    Yield(func() error {accessCount[k] = v; return nil}).
//		    Run(ctx)
func (f *fetcher) ScanInto(ptrs ...any) *fetcher {
	f.dst = ptrs
	return f
}

// Yield sets a func that is called once for each row.
//
// Use it with ScanInto (see example there).
func (f *fetcher) Yield(yield func() error) *fetcher {
	f.yield = yield
	return f
}

// YieldColumns is like Yield but will get a slice of pointers to column values each row.
// Do not change the slice contents, it must only ever be read.
// YieldColumns is less efficient than yield.
func (f *fetcher) YieldColumns(yield func([]any) error) *fetcher {
	f.yield = func() error {
		return yield(f.dst)
	}
	return f
}

// HandleColumns receives a function that will be called on results before the first
// yield is called.
// The func cols will receive the result of database/sql:Rows.ColumnTypes().
// If an error is reported, the whole query will be cancelled.
//
// When used with MySQL, f.Prepared(true) should be used if you intend to use numeric types.
// MySQL only uses a typed result in binary protocol, which is only used with prepared statements.
// Text protocol only returns string values as sql.RawBytes.
func (f *fetcher) InitColumns(initCols func([]*sql.ColumnType, error) error) *fetcher {
	// requires f.prepared = true for MySQL
	f.initCols = initCols
	return f
}

// Run the query.
func (f *fetcher) Run(ctx context.Context, args ...any) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if f.initCols == nil && f.dst == nil {
		// derive scan types just before rows.Scan
		f.initCols = f.deriveScan()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var rows *sql.Rows
	if f.asStmt {
		var stmt *sql.Stmt
		stmt, err = f.db.PrepareContext(ctx, f.query)
		if err != nil {
			err = querror{f.query, err}
			return
		}
		defer stmt.Close()
		rows, err = stmt.QueryContext(ctx, args...)
	} else {
		rows, err = f.db.QueryContext(ctx, f.query, args...)
	}
	if err != nil {
		err = querror{f.query, err}
		return err
	}
	defer func() {
		cerr := rows.Close()
		if err == nil {
			err = cerr
		}
	}()
	if f.initCols != nil {
		// for MySQL this should be used with f.Prepared(true)
		err = f.initCols(rows.ColumnTypes())
		if err != nil {
			err = querror{f.query, err}
			return err
		}
	}
	for rows.Next() {
		err = rows.Scan(f.dst...)
		if err != nil {
			return err
		}
		if f.yield != nil {
			err = f.yield()
			if err != nil {
				return err
			}
		}
	}
	err = rows.Err()
	return err
}
