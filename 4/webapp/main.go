package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"

	"github.com/go-martini/martini"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
)

var db *sql.DB
var (
	store     = sessions.NewCookieStore([]byte("secret-isucon"))
	templates = template.Must(template.ParseFiles("templates/index.tmpl", "templates/mypage.tmpl"))
)
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

	m.Get("/", func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "isucon_go_session")
		notice := getFlash(session, "notice")

		session.Save(r, w)

		templates.ExecuteTemplate(w, "index.tmpl", map[string]string{"Flash": notice})
	})

	m.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "isucon_go_session")
		user, err := attemptLogin(r)

		if err != nil || user == nil {
			notice := ""
			switch err {
			case ErrBannedIP:
				notice = "You're banned."
			case ErrLockedUser:
				notice = "This account is locked."
			default:
				notice = "Wrong username or password"
			}

			session.Values["notice"] = notice
			session.Save(r, w)

			http.Redirect(w, r, "/", 302)
			return
		}

		session.Values["user_id"] = strconv.Itoa(user.ID)
		session.Save(r, w)
		http.Redirect(w, r, "/mypage", 302)
	})

	m.Get("/mypage", func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "isucon_go_session")

		userID, ok := session.Values["user_id"]
		if !ok {
			session.Values["notice"] = "You must be logged in"
			session.Save(r, w)
			http.Redirect(w, r, "/", 302)
			return
		}

		templates.ExecuteTemplate(w, "mypage.tmpl", getLastLogin(userID))
	})

	m.Get("/report", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string][]string{
			"banned_ips":   bannedIPs(),
			"locked_users": lockedUsers(),
		})
	})

	log.Fatal(unixSocketServe("/tmp/isucon_go.sock", m))
	// log.Fatal(http.ListenAndServe(":8081", m))
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
