package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"

	"github.com/go-martini/martini"
	_ "github.com/go-sql-driver/mysql"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
)

var db *sql.DB
var (
	userLockThreshold int
	iPBanThreshold    int
)

func init() {
	dsn := fmt.Sprintf(
		"%s:%s@unix(/var/run/mysqld/mysqld.sock)/%s?parseTime=true&loc=Local",
		getEnv("ISU4_DB_USER", "root"),
		getEnv("ISU4_DB_PASSWORD", ""),
		getEnv("ISU4_DB_NAME", "isu4_qualifier"),
	)

	var err error

	db, err = sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	db.SetMaxIdleConns(100)

	userLockThreshold, err = strconv.Atoi(getEnv("ISU4_USER_LOCK_THRESHOLD", "3"))
	if err != nil {
		panic(err)
	}

	iPBanThreshold, err = strconv.Atoi(getEnv("ISU4_IP_BAN_THRESHOLD", "10"))
	if err != nil {
		panic(err)
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	m := martini.Classic()

	store := sessions.NewCookieStore([]byte("secret-isucon"))
	m.Use(sessions.Sessions("isucon_go_session", store))

	m.Use(render.Renderer(render.Options{
		Layout: "layout",
	}))

	m.Get("/", func(r render.Render, session sessions.Session) {
		r.HTML(200, "index", map[string]string{"Flash": getFlash(session, "notice")})
	})

	m.Post("/login", func(req *http.Request, r render.Render, session sessions.Session) {
		user, err := attemptLogin(req)

		notice := ""
		if err != nil || user == nil {
			switch err {
			case ErrBannedIP:
				notice = "You're banned."
			case ErrLockedUser:
				notice = "This account is locked."
			default:
				notice = "Wrong username or password"
			}

			session.Set("notice", notice)
			r.Redirect("/")
			return
		}

		session.Set("user_id", strconv.Itoa(user.ID))
		r.Redirect("/mypage")
	})

	m.Get("/mypage", func(r render.Render, session sessions.Session) {
		userID := session.Get("user_id")

		if userID == nil {
			session.Set("notice", "You must be logged in")
			r.Redirect("/")
			return
		}

		r.HTML(200, "mypage", getLastLogin(userID))
	})

	m.Get("/report", func(r render.Render) {
		r.JSON(200, map[string][]string{
			"banned_ips":   bannedIPs(),
			"locked_users": lockedUsers(),
		})
	})

	log.Fatal(unixSocketServe("/tmp/isucon_go.sock", m))
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
