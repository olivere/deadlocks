# MySQL vs PostgreSQL

```sh
$ docker-compose up -d
$ make build
go build
$ ls deadlocks
deadlocks
```

See how MySQL fails ...

```sh
$ make mysql
.EEE.EE.E. ✗
$ make mysql
EEEEE..EE. ✗
$ make mysql
...EE.E... ✗
```

... and PostgreSQL succeeds:

```sh
$ make postgres
.......... ✔
$ make postgres
.......... ✔
$ make postgres
.......... ✔
```

Now, start MySQL with retry on failure (will retry only on
`Error 1213: Deadlock found when trying to get lock; try restarting transaction`):

```sh
$ make mysql-retry
.......... ✔
$ make mysql-retry
.......... ✔
$ make mysql-retry
.......... ✔
```

PostgreSQL, again, will make no difference:

```sh
$ make postgres-retry
.......... ✔
$ make postgres-retry
.......... ✔
$ make postgres-retry
.......... ✔
```

## Prior art

This code just changes [github.com/mlomnicki/mysql-vs-postgres-deadlock](https://github.com/mlomnicki/mysql-vs-postgres-deadlock) slightly.
