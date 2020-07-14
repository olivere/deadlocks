.PHONY: build mysql mysql-retry postgres postgres-retry

build:
	go build

mysql:
	./deadlocks -driver=mysql -dsn="go:go@tcp(localhost)/deadlocks?multiStatements=true" -concurrency=10

mysql-retry:
	./deadlocks -driver=mysql -dsn="go:go@tcp(localhost)/deadlocks?multiStatements=true" -concurrency=10 -retry=true

postgres:
	./deadlocks -driver=postgres -dsn="postgres://go:go@localhost/deadlocks?sslmode=disable" -concurrency=10

postgres-retry:
	./deadlocks -driver=postgres -dsn="postgres://go:go@localhost/deadlocks?sslmode=disable" -concurrency=10 -retry=true
