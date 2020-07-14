package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/olivere/deadlocks/tx"
)

const defaultConcurrencyLevel = 30

var (
	driver           = flag.String("driver", envString("mysql", "DRIVER"), "Connection string")
	dsn              = flag.String("dsn", envString("", "DSN"), "Connection string")
	concurrencyLevel = flag.Int("concurrency", envInt(defaultConcurrencyLevel, "CONCURRENCY"), "Conncurrency level")
	retry            = flag.Bool("retry", false, "Retry transactions")
)

const MySQL_SCHEMA = `
    DROP TABLE IF EXISTS comments;
    DROP TABLE IF EXISTS posts;
    CREATE TABLE posts (
      id             INT(11) NOT NULL AUTO_INCREMENT,
      comments_count INT(11) NOT NULL,
      PRIMARY KEY (id)
    );
    CREATE TABLE comments (
      id      INT(11) NOT NULL AUTO_INCREMENT,
      post_id INT(11) NOT NULL,
      PRIMARY KEY (id),
      FOREIGN KEY (post_id) REFERENCES posts(id)
    );
    INSERT INTO posts (id, comments_count) VALUES (1, 0);
`

const POSTGRES_SCHEMA = `
  DROP TABLE IF EXISTS comments;
  DROP TABLE IF EXISTS posts;
  CREATE TABLE posts (
    id             serial PRIMARY KEY,
    comments_count integer
  );
  CREATE TABLE comments (
    id      serial PRIMARY KEY,
    post_id integer REFERENCES posts
  );
  INSERT INTO posts (id, comments_count) VALUES (1, 0);
`

const DEADLOCKING_STATEMENT = `
	BEGIN;
  INSERT INTO comments (post_id) VALUES (1);
  UPDATE posts SET comments_count = comments_count + 1 WHERE id = 1;
	COMMIT;
`

const COMMENTS_COUNT_QUERY = `
	SELECT comments_count FROM posts WHERE id = 1;
`

func simulateDeadlock(ctx context.Context, db *sql.DB, fn func(ctx context.Context, db *sql.DB, tx *sql.Tx) error) (err error) {
	var wg sync.WaitGroup
	start := make(chan struct{})
	wg.Add(*concurrencyLevel)
	for i := 0; i < *concurrencyLevel; i++ {
		go func() {
			<-start
			if *retry {
				err = tx.RunWithRetry(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
					ie := fn(ctx, db, tx)
					if ie != nil {
						if !IsMySQLDeadlock(ie) {
							log.Fatal(ie)
						}
						fmt.Print("E\033[1D")
					} else {
						fmt.Print(".\033[1D")
					}
					return ie
				})
			} else {
				err = fn(ctx, db, nil)
				if err != nil && !IsMySQLDeadlock(err) {
					log.Fatal(err)
				}
			}
			if err != nil {
				fmt.Print("E")
			} else {
				fmt.Print(".")
			}
			wg.Done()
		}()
	}
	close(start)
	wg.Wait()
	return
}

// -- MySQL --

func runMySQL(dsn string) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := execMySQL(context.Background(), db, nil, MySQL_SCHEMA); err != nil {
		log.Fatal(err)
	}
	_ = simulateDeadlock(
		context.Background(),
		db,
		func(ctx context.Context, db *sql.DB, tx *sql.Tx) error {
			return execMySQL(ctx, db, tx, DEADLOCKING_STATEMENT)
		},
	)

	row := db.QueryRow(COMMENTS_COUNT_QUERY)
	var commentsCount int
	if err := row.Scan(&commentsCount); err != nil {
		log.Fatal(err)
	}
	if commentsCount != *concurrencyLevel {
		fmt.Printf(" \033[31m✗\033[32m\n")
	} else {
		fmt.Printf(" \033[32m✔\033[32m\n")
	}
}

func execMySQL(ctx context.Context, db *sql.DB, tx *sql.Tx, statement string) (err error) {
	if tx != nil {
		_, err = tx.ExecContext(ctx, statement)
	} else {
		_, err = db.ExecContext(ctx, statement)
	}
	return
}

// -- PostgreSQL --

func runPostgres(dsn string) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(POSTGRES_SCHEMA)
	if err != nil {
		log.Fatal(err)
	}
	simulateDeadlock(
		context.Background(),
		db,
		func(ctx context.Context, db *sql.DB, tx *sql.Tx) error {
			return execPostgres(ctx, db, tx, DEADLOCKING_STATEMENT)
		},
	)

	row := db.QueryRow(COMMENTS_COUNT_QUERY)
	var commentsCount int
	if err := row.Scan(&commentsCount); err != nil {
		log.Fatal(err)
	}
	if commentsCount != *concurrencyLevel {
		fmt.Printf(" \033[31m✗\033[32m\n")
	} else {
		fmt.Printf(" \033[32m✔\033[32m\n")
	}
}

func execPostgres(ctx context.Context, db *sql.DB, tx *sql.Tx, statement string) (err error) {
	if tx != nil {
		_, err = tx.ExecContext(ctx, statement)
	} else {
		_, err = db.ExecContext(ctx, statement)
	}
	return
}

// -- main --

func usage() {
	w := flag.CommandLine.Output()
	fmt.Fprintf(w, "Usage: %s -driver=<mysql|postgres> -dsn=... -concurrency=<n> -retry=<true|false>\n", os.Args[0])
	fmt.Fprintln(w)
	flag.PrintDefaults()
	fmt.Fprintln(w)
	os.Exit(1)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Usage = usage
	flag.Parse()

	switch *driver {
	case "mysql":
		runMySQL(*dsn)
	case "postgres":
		runPostgres(*dsn)
	default:
		usage()
	}
}

// -- Helpers --

func IsMySQLDeadlock(err error) bool {
	if e, ok := err.(*mysql.MySQLError); ok {
		// Error 1213: Deadlock found when trying to get lock; try restarting transaction
		return e.Number == 1213
	}
	return false
}

func envString(defaultValue string, keys ...string) string {
	for _, key := range keys {
		if s := os.Getenv(key); s != "" {
			return s
		}
	}
	return defaultValue
}

func envInt(defaultValue int, keys ...string) int {
	for _, key := range keys {
		if s := os.Getenv(key); s != "" {
			if v, err := strconv.ParseInt(s, 10, 64); err == nil {
				return int(v)
			}
		}
	}
	return defaultValue
}
