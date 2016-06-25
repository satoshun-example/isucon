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

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
)

var (
	db          *sql.DB
	failUserIds map[int]int
)

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

	failUserIds = make(map[int]int)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	// POST
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "isucon_go_session")
		user, err := attemptLogin(r)

		if err != nil || user == nil {
			notice := ""
			switch err {
			case ErrBannedIP:
				notice = "banned"
			case ErrLockedUser:
				notice = "locked"
			default:
				notice = "wrong"
			}

			http.SetCookie(w, &http.Cookie{Name: "notice", Value: notice})
			http.Redirect(w, r, "/", 302)
			return
		}

		session.Values["user_id"] = strconv.Itoa(user.ID)
		session.Save(r, w)
		http.Redirect(w, r, "/mypage", 302)
	})

	http.HandleFunc("/mypage", func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "isucon_go_session")

		userID, ok := session.Values["user_id"]
		if !ok {
			http.SetCookie(w, &http.Cookie{Name: "notice", Value: "logged"})
			http.Redirect(w, r, "/", 302)
			return
		}

		templates.ExecuteTemplate(w, "mypage.tmpl", getLastLogin(userID))
	})

	http.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string][]string{
			"banned_ips":   bannedIPs(),
			"locked_users": lockedUsers(),
		})
	})

	log.Fatal(unixSocketServe("/tmp/isucon_go.sock", nil))
	// log.Fatal(http.ListenAndServe(":8081", nil))
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
