package main

import (
	"database/sql"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/garyburd/redigo/redis"
)

func createDB() (*sql.DB, error) {
	host := os.Getenv("ISUCON5_DB_HOST")
	if host == "" {
		host = "localhost"
	}
	portstr := os.Getenv("ISUCON5_DB_PORT")
	if portstr == "" {
		portstr = "3306"
	}
	user := os.Getenv("ISUCON5_DB_USER")
	if user == "" {
		user = "root"
	}
	password := os.Getenv("ISUCON5_DB_PASSWORD")
	dbname := os.Getenv("ISUCON5_DB_NAME")
	if dbname == "" {
		dbname = "isucon5q"
	}

	// db, err = sql.Open("mysql", user+":"+password+"@tcp("+host+":"+strconv.Itoa(port)+")/"+dbname+"?loc=Local&parseTime=true")
	return sql.Open("mysql", user+":"+password+"@unix(/var/run/mysqld/mysqld.sock)/"+dbname+"?loc=Local&parseTime=true")
}

func newRedisPool(server string, idle int) *redis.Pool {
	return &redis.Pool{
		MaxIdle: idle,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("unix", server)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func unixSocketServe(path string, handler http.Handler) error {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		os.Remove(path)
	}

	listener, err := net.ListenUnix("unix", &net.UnixAddr{path, "unix"})
	if err != nil {
		panic(err)
	}

	if err := os.Chmod(path, 0777); err != nil {
		panic(err)
	}

	return http.Serve(listener, handler)
}
