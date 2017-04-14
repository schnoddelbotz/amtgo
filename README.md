# amtgo [![Build Status](https://travis-ci.org/schnoddelbotz/amtgo.svg?branch=master)](https://travis-ci.org/schnoddelbotz/amtgo) [![GoDoc](https://godoc.org/github.com/schnoddelbotz/amtgo/amt?status.png)](https://godoc.org/github.com/schnoddelbotz/amtgo/amt)

A [golang](https://golang.org/) implementation of
[amtc](https://github.com/schnoddelbotz/amtc) and its amtc-web GUI.
Like amtc -- for mass remote power management and monitoring [Intel AMT](http://en.wikipedia.org/wiki/Intel_Active_Management_Technology) hosts.

Features over amtc / amtc-web:

- [x] includes amtc-web application webserver in a single binary
- [x] instant, parallel execution of GUI-submitted tasks
- [x] includes cron functionality for monitoring and scheduled tasks
- [x] it finally has a command/context-sensitive `-h(elp)` command line switch
- [x] windows binaries are available on [releases](./../../releases) page, too

amtgo still supports SQLite and MySQL as database back-ends.
amtc-web GUI is included 1:1 from amtc repository.

## binary installation

Download the latest [release](./../../releases) for your OS.
Unzip the download -- the zip file contains the amtgo binary.
Move the binary to a location of your choice, e.g. `/usr/local/bin`.
That's all needed to run amtgo from CLI like amtc.
You should be able to run `amtgo -h` for help.

`amtgo server` should be started up via systemd or the like.
Find first run instructions below.

### ... when using SQLite

 - In a terminal / CMD session, `cd` to the directory where
   amtgo is supposed to store its SQLite database and
   AMT password files.
 - Put your AMT password into a file, e.g. `echo 'mY$ecurePassw0rd' > amtpassword.txt`
 - Type/Run `amtgo server`. Visit http://localhost:8080 to create initial
   web site user account and log in

### ... when using MySQL

 - Install MySQL, start it and create a database and a user for amtgo purposes.
 - Proceed like with SQLite above
 - Starting amtgo server using MySQL requires MySQL connection details.
   They should be passed as environment variables, but can also be passed
   as command line arguments for testing. Output excerpt from `amtgo server -h`:

```
OPTIONS:
   --dbdriver value, -d value    Database driver: sqlite3 or mysql (default: "sqlite3") [$DB_DRIVER]
   --dbfile value, -F value      SQLite database file (default: "amtgo.db") [$DB_FILE]
   --dbName value, -D value      MySQL database name (default: "amtgo") [$DB_NAME]
   --dbHost value, -H value      MySQL database host (default: "localhost") [$DB_HOST]
   --dbUser value, -U value      MySQL database user name (default: "amtgo") [$DB_USER]
   --dbPassword value, -P value  MySQL database password [$DB_PASSWORD]
   --dbPort value, -p value      MySQL database port (default: "3306") [$DB_PORT]
```

For example, to run amtgo server using mysql database `foo` on localhost as user `joe`
using password `1234`, run: `amtgo server -d mysql -D foo -U joe -P 1234`.

For testing purposes, you may want to run a ephemeral MySQL instance using Docker:

```bash
docker run --rm --name amtgo-mysql \
  -e MYSQL_RANDOM_ROOT_PASSWORD=yes \
  -e MYSQL_DATABASE=amtgo \
  -e MYSQL_USER=amtgo \
  -e MYSQL_PASSWORD=1234 \
  -p 3306:3306 \
  -d mysql/mysql-server
```

## building amtgo from source

```bash
# if using Go < 1.8
export GOPATH=/home/you/go

# fetch and build amtgo
go get -v github.com/schnoddelbotz/amtgo
```

To run tests, git clone the repository and run `make test` or `make coverage`.
Note that the repository contains a pre-built amtc-web; to rebuild amtc-web
from source, run `make assets`, then `make` to build amtgo itself.

## status & open issues

amtgo is a fun project for me to get into golang -- I never was too
happy with amtc-web's requirement for Apache, PHP and cron...
Like amtc, amtgo works for me (tm), but I lack long-term usage experience.
Note that amtgo, in contrast to amtc, does not support EOI protocol but
only WS-MAN. For AMT versions <6, you might want to stay with amtc.

Please report any [issues](./../../issues/) you encounter.
I'm also thankful for any hints for improvement.

## license

amtgo is published under [MIT license](LICENSE.txt).

Besides golang's standard library, amtgo relies on:

- [urfave/cli](https://github.com/urfave/cli) for CLI parsing
- [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) as SQLite driver
- [jmoiron/sqlx](https://github.com/jmoiron/sqlx) as extensions to golang's database/sql package
- [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) as MySQL driver
- [gorilla](https://github.com/gorilla) handlers, mux, securecookie & sessions for amtc-web
- [xinsnake/go-http-digest-auth-client](https://github.com/xinsnake/go-http-digest-auth-client),
  [tweaked](tree/master/amt/digest_auth_client) to support TLS, timeouts and certificate
  verification for AMT.

Assets (i.e. amtc-web static content and AMT commands) are included
using [go-bindata](https://github.com/jteeuwen/go-bindata).
Releases for all platforms are built using [xgo](https://github.com/karalabe/xgo).

Please check [amtc-web 3rd party components](https://github.com/schnoddelbotz/amtc/blob/master/amtc-web/LICENSES-3rd-party.txt), too.
