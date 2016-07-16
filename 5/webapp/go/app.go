package main

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

var (
	db    *sql.DB
	pool  *redis.Pool
	store *sessions.CookieStore

	templates = template.Must(template.New("app").Funcs(template.FuncMap{
		"substring": func(s string, l int) string {
			if len(s) > l {
				return s[:l]
			}
			return s
		},
		"split": strings.Split,
	}).ParseGlob("templates/*.html"))
)

type User struct {
	ID          int
	AccountName string
	NickName    string
	Email       string
}

type Profile struct {
	UserID    int
	FirstName string
	LastName  string
	Sex       string
	Birthday  mysql.NullTime
	Pref      string
	UpdatedAt time.Time
}

type Entry struct {
	ID        int
	UserID    int
	Private   bool
	Title     string
	Content   string
	CreatedAt time.Time
}

type Comment struct {
	ID        int
	EntryID   int
	UserID    int
	Comment   string
	CreatedAt time.Time
}

type Friend struct {
	ID        int
	CreatedAt time.Time
}

type Footprint struct {
	UserID    int
	OwnerID   int
	CreatedAt time.Time
	Updated   time.Time
}

var prefs = []string{"未入力",
	"北海道", "青森県", "岩手県", "宮城県", "秋田県", "山形県", "福島県", "茨城県", "栃木県", "群馬県", "埼玉県", "千葉県", "東京都", "神奈川県", "新潟県", "富山県",
	"石川県", "福井県", "山梨県", "長野県", "岐阜県", "静岡県", "愛知県", "三重県", "滋賀県", "京都府", "大阪府", "兵庫県", "奈良県", "和歌山県", "鳥取県", "島根県",
	"岡山県", "広島県", "山口県", "徳島県", "香川県", "愛媛県", "高知県", "福岡県", "佐賀県", "長崎県", "熊本県", "大分県", "宮崎県", "鹿児島県", "沖縄県"}

var (
	ErrAuthentication   = errors.New("Authentication error.")
	ErrPermissionDenied = errors.New("Permission denied.")
	ErrContentNotFound  = errors.New("Content not found.")
)

func authenticate(w http.ResponseWriter, r *http.Request, email, passwd string) {
	query := `
SELECT u.id AS id, u.account_name AS account_name, u.nick_name AS nick_name, u.email AS email
FROM users u
WHERE u.email = ? AND u.passhash = SHA2(CONCAT(?, u.salt), 512)`
	row := db.QueryRow(query, email, passwd)
	user := User{}
	err := row.Scan(&user.ID, &user.AccountName, &user.NickName, &user.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			checkErr(ErrAuthentication)
		}
		checkErr(err)
	}
	session := getSession(w, r)
	session.Values["user_id"] = user.ID
	session.Save(r, w)
}

func getCurrentUser(w http.ResponseWriter, r *http.Request) *User {
	u := context.Get(r, "user")
	if u != nil {
		user := u.(User)
		return &user
	}
	session := getSession(w, r)
	userID, ok := session.Values["user_id"]
	if !ok || userID == nil {
		return nil
	}
	row := db.QueryRow(`SELECT id, account_name, nick_name, email FROM users WHERE id=?`, userID)
	user := User{}
	err := row.Scan(&user.ID, &user.AccountName, &user.NickName, &user.Email)
	if err == sql.ErrNoRows {
		checkErr(ErrAuthentication)
	}
	checkErr(err)
	context.Set(r, "user", user)
	return &user
}

func authenticated(w http.ResponseWriter, r *http.Request) *User {
	user := getCurrentUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return nil
	}
	return user
}

func getUser(w http.ResponseWriter, userID int) *User {
	row := db.QueryRow(`SELECT id, account_name, nick_name, email FROM users WHERE id = ?`, userID)
	user := User{}
	err := row.Scan(&user.ID, &user.AccountName, &user.NickName, &user.Email)
	if err == sql.ErrNoRows {
		checkErr(ErrContentNotFound)
	}
	checkErr(err)
	return &user
}

func getUserFromAccount(w http.ResponseWriter, name string) *User {
	row := db.QueryRow(`SELECT id, account_name, nick_name, email FROM users WHERE account_name = ?`, name)
	user := User{}
	err := row.Scan(&user.ID, &user.AccountName, &user.NickName, &user.Email)
	if err == sql.ErrNoRows {
		checkErr(ErrContentNotFound)
	}
	checkErr(err)
	return &user
}

func isFriend(w http.ResponseWriter, r *http.Request, anotherID int) bool {
	session := getSession(w, r)
	id := session.Values["user_id"]
	row := db.QueryRow(`SELECT COUNT(1) AS cnt FROM relations WHERE (one = ? AND another = ?)`, id, anotherID)
	cnt := new(int)
	err := row.Scan(cnt)
	checkErr(err)
	return *cnt > 0
}

func isFriendAccount(w http.ResponseWriter, r *http.Request, name string) (*User, bool) {
	user := getUserFromAccount(w, name)
	if user == nil {
		return nil, false
	}
	return user, isFriend(w, r, user.ID)
}

func permitted(w http.ResponseWriter, r *http.Request, userID int, anotherID int) bool {
	if anotherID == userID {
		return true
	}
	return isFriend(w, r, anotherID)
}

func markFootprint(user *User, id int) {
	if user != nil && user.ID != id {
		_, err := db.Exec(`INSERT INTO footprints (user_id, owner_id) VALUES (?,?)`, id, user.ID)
		checkErr(err)
	}
}

func myHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rcv := recover()
			if rcv != nil {
				switch {
				case rcv == ErrAuthentication:
					session := getSession(w, r)
					delete(session.Values, "user_id")
					session.Save(r, w)
					render(w, r, http.StatusUnauthorized, "login.html", struct{ Message string }{"ログインに失敗しました"})
					return
				case rcv == ErrPermissionDenied:
					render(w, r, http.StatusForbidden, "error.html", struct{ Message string }{"友人のみしかアクセスできません"})
					return
				case rcv == ErrContentNotFound:
					render(w, r, http.StatusNotFound, "error.html", struct{ Message string }{"要求されたコンテンツは存在しません"})
					return
				default:
					var msg string
					if e, ok := rcv.(runtime.Error); ok {
						msg = e.Error()
					}
					if s, ok := rcv.(string); ok {
						msg = s
					}
					msg = rcv.(error).Error()
					http.Error(w, msg, http.StatusInternalServerError)
				}
			}
		}()
		fn(w, r)
	}
}

func getSession(w http.ResponseWriter, r *http.Request) *sessions.Session {
	session, _ := store.Get(r, "isucon5q-go.session")
	return session
}

func getTemplatePath(file string) string {
	return path.Join("templates", file)
}

func render(w http.ResponseWriter, r *http.Request, status int, file string, data interface{}) {
	w.WriteHeader(status)
	checkErr(templates.ExecuteTemplate(w, file, data))
}

func GetLogin(w http.ResponseWriter, r *http.Request) {
	render(w, r, http.StatusOK, "login.html", struct{ Message string }{"高負荷に耐えられるSNSコミュニティサイトへようこそ!"})
}

func PostLogin(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	passwd := r.FormValue("password")
	authenticate(w, r, email, passwd)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func GetLogout(w http.ResponseWriter, r *http.Request) {
	session := getSession(w, r)
	delete(session.Values, "user_id")
	session.Options = &sessions.Options{MaxAge: -1}
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

type IComment struct {
	AccountName string
	NickName    string
	Comment     string
	CreatedAt   time.Time
}

type IFootPrint struct {
	AccountName string
	NickName    string
	CreatedAt   time.Time
}

type IEntry struct {
	ID          int
	Title       string
	AccountName string
	NickName    string
	CreatedAt   time.Time
}

type FriendComment struct {
	ID          int
	EntryID     int
	UserID      int
	Comment     string
	EntryUserID int
	CreatedAt   time.Time
}

type FriendCommentData struct {
	NickName    string
	AccountName string
}

func GetIndex(w http.ResponseWriter, r *http.Request) {
	user := authenticated(w, r)
	if user == nil {
		return
	}

	var wg sync.WaitGroup

	wg.Add(1)
	prof := Profile{}
	go func() {
		defer wg.Done()

		row := db.QueryRow(`
SELECT * FROM profiles
WHERE user_id = ?`, user.ID)
		err := row.Scan(&prof.UserID, &prof.FirstName, &prof.LastName, &prof.Sex, &prof.Birthday, &prof.Pref, &prof.UpdatedAt)
		if err != sql.ErrNoRows {
			checkErr(err)
		}
	}()

	wg.Add(1)
	entries := make([]Entry, 0, 5)
	go func() {
		defer wg.Done()

		rows, err := db.Query(`
SELECT id, body FROM entries
WHERE user_id = ?
ORDER BY created_at LIMIT 5`, user.ID)
		defer rows.Close()
		if err != sql.ErrNoRows {
			checkErr(err)
		}

		for rows.Next() {
			var id int
			var body string
			checkErr(rows.Scan(&id, &body))
			entries = append(entries, Entry{ID: id, Title: strings.SplitN(body, "\n", 2)[0]})
		}
	}()

	wg.Add(1)
	commentsForMe := make([]IComment, 0, 10)
	go func() {
		defer wg.Done()

		rows, err := db.Query(`
SELECT c.comment AS comment, c.created_at AS created_at, u.nick_name AS nick_name, u.account_name AS account_name
FROM comments c
JOIN entries e ON c.entry_id = e.id
JOIN users u ON u.id = e.user_id
WHERE e.user_id = ?
ORDER BY c.created_at DESC
LIMIT 10`, user.ID)

		defer rows.Close()

		if err != sql.ErrNoRows {
			checkErr(err)
		}

		for rows.Next() {
			c := IComment{}
			checkErr(rows.Scan(&c.Comment, &c.CreatedAt, &c.NickName, &c.AccountName))
			commentsForMe = append(commentsForMe, c)
		}
	}()

	wg.Add(1)
	entriesOfFriends := make([]IEntry, 0, 10)
	go func() {
		defer wg.Done()

		rows, err := db.Query(`
SELECT e.id, e.body, e.created_at, u.account_name, u.nick_name FROM relations r
INNER JOIN (SELECT id, body, user_id, created_at FROM entries
	ORDER BY created_at DESC
	LIMIT 1000) as e ON e.user_id = r.one
INNER JOIN users u ON e.user_id = u.id
WHERE r.another = ?
ORDER BY e.created_at DESC
LIMIT 10`, user.ID)
		if err != sql.ErrNoRows {
			checkErr(err)
		}

		for rows.Next() {
			var id int
			var body, accountName, nickName string
			var createdAt time.Time
			checkErr(rows.Scan(&id, &body, &createdAt, &accountName, &nickName))

			entriesOfFriends = append(entriesOfFriends,
				IEntry{
					ID:          id,
					Title:       strings.SplitN(body, "\n", 2)[0],
					CreatedAt:   createdAt,
					AccountName: accountName,
					NickName:    nickName,
				})
		}
		rows.Close()
	}()

	wg.Add(1)
	commentsOfFriends := make([]FriendComment, 0, 10)
	commentsOfFriendsData := make(map[int]FriendCommentData)
	go func() {
		defer wg.Done()

		rows, err := db.Query(`
SELECT c.entry_id, c.user_id, c.comment, c.created_at, e.user_id
FROM (SELECT entry_id, user_id, comment, created_at FROM comments
      ORDER BY created_at
      DESC LIMIT 1000) as c
INNER JOIN relations r ON r.one = c.user_id
INNER JOIN entries e ON e.id = c.entry_id
WHERE r.another = ? AND
	  (e.private = 0 OR EXISTS (SELECT * FROM relations rr WHERE rr.one = c.user_id AND rr.another = e.user_id))
ORDER BY created_at DESC
LIMIT 10`, user.ID)

		if err != sql.ErrNoRows {
			checkErr(err)
		}

		ids := make(map[int]struct{})
		for rows.Next() {
			c := FriendComment{}
			checkErr(rows.Scan(
				&c.EntryID, &c.UserID, &c.Comment, &c.CreatedAt,
				&c.EntryUserID))

			commentsOfFriends = append(commentsOfFriends, c)

			ids[c.UserID] = struct{}{}
			ids[c.EntryUserID] = struct{}{}
		}

		rows.Close()

		keys := make([]string, 0, len(ids))
		for key := range ids {
			keys = append(keys, strconv.Itoa(key))
		}

		rows, err = db.Query(fmt.Sprintf(`
SELECT id, nick_name, account_name
FROM users
WHERE id IN (%s)`, strings.Join(keys, ",")))

		if err != sql.ErrNoRows {
			checkErr(err)
		}

		for rows.Next() {
			var id int
			var data FriendCommentData
			checkErr(rows.Scan(&id, &data.NickName, &data.AccountName))
			commentsOfFriendsData[id] = data
		}

		rows.Close()
	}()

	wg.Add(1)
	friends := 0
	go func() {
		defer wg.Done()
		// count friends
		row := db.QueryRow(`
SELECT COUNT(*) FROM relations
WHERE one = ?`, user.ID)
		row.Scan(&friends)
	}()

	wg.Add(1)
	footprints := make([]IFootPrint, 0, 10)
	go func() {
		defer wg.Done()
		rows, err := db.Query(`
SELECT DATE(f.created_at) AS date, MAX(f.created_at) AS updated, MIN(u.account_name), MIN(u.nick_name)
FROM footprints f
INNER JOIN users u ON u.id = f.owner_id
WHERE f.user_id = ?
GROUP BY user_id, f.owner_id, DATE(f.created_at)
ORDER BY updated DESC
LIMIT 10`, user.ID)
		if err != sql.ErrNoRows {
			checkErr(err)
		}
		for rows.Next() {
			fp := IFootPrint{}
			checkErr(rows.Scan(&fp.CreatedAt, &time.Time{}, &fp.AccountName, &fp.NickName))
			footprints = append(footprints, fp)
		}
		rows.Close()
	}()

	wg.Wait()
	render(w, r, http.StatusOK, "index.html", struct {
		User                  User
		Profile               Profile
		Entries               []Entry
		CommentsForMe         []IComment
		EntriesOfFriends      []IEntry
		CommentsOfFriends     []FriendComment
		CommentsOfFriendsData map[int]FriendCommentData
		Friends               int
		Footprints            []IFootPrint
	}{
		*user, prof, entries, commentsForMe, entriesOfFriends, commentsOfFriends, commentsOfFriendsData, friends, footprints,
	})
}

func GetProfile(w http.ResponseWriter, r *http.Request) {
	user := authenticated(w, r)
	if user == nil {
		return
	}

	account := mux.Vars(r)["account_name"]
	owner := getUserFromAccount(w, account)
	row := db.QueryRow(`SELECT * FROM profiles WHERE user_id = ?`, owner.ID)
	prof := Profile{}
	err := row.Scan(&prof.UserID, &prof.FirstName, &prof.LastName, &prof.Sex, &prof.Birthday, &prof.Pref, &prof.UpdatedAt)
	if err != sql.ErrNoRows {
		checkErr(err)
	}
	var query string
	if permitted(w, r, user.ID, owner.ID) {
		query = `SELECT * FROM entries WHERE user_id = ? ORDER BY created_at LIMIT 5`
	} else {
		query = `SELECT * FROM entries WHERE user_id = ? AND private=0 ORDER BY created_at LIMIT 5`
	}
	rows, err := db.Query(query, owner.ID)
	if err != sql.ErrNoRows {
		checkErr(err)
	}
	entries := make([]Entry, 0, 5)
	for rows.Next() {
		var id, userID, private int
		var body string
		var createdAt time.Time
		checkErr(rows.Scan(&id, &userID, &private, &body, &createdAt))
		entry := Entry{id, userID, private == 1, strings.SplitN(body, "\n", 2)[0], strings.SplitN(body, "\n", 2)[1], createdAt}
		entries = append(entries, entry)
	}
	rows.Close()

	markFootprint(user, owner.ID)

	render(w, r, http.StatusOK, "profile.html", struct {
		Owner       User
		Profile     Profile
		Entries     []Entry
		Private     bool
		User        *User
		Prefectures []string
	}{
		*owner, prof, entries, permitted(w, r, user.ID, owner.ID), user, prefs,
	})
}

func PostProfile(w http.ResponseWriter, r *http.Request) {
	user := authenticated(w, r)
	if user == nil {
		return
	}
	account := mux.Vars(r)["account_name"]
	if account != user.AccountName {
		checkErr(ErrPermissionDenied)
	}
	query := `
UPDATE profiles
SET first_name=?, last_name=?, sex=?, birthday=?, pref=?, updated_at=CURRENT_TIMESTAMP()
WHERE user_id = ?`
	birth := r.FormValue("birthday")
	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")
	sex := r.FormValue("sex")
	pref := r.FormValue("pref")
	_, err := db.Exec(query, firstName, lastName, sex, birth, pref, user.ID)
	checkErr(err)
	// TODO should escape the account name?
	http.Redirect(w, r, "/profile/"+account, http.StatusSeeOther)
}

type LEntry struct {
	ID        int
	Private   bool
	Title     string
	Content   string
	Count     int
	CreatedAt time.Time
}

func ListEntries(w http.ResponseWriter, r *http.Request) {
	user := authenticated(w, r)
	if user == nil {
		return
	}

	account := mux.Vars(r)["account_name"]
	owner := getUserFromAccount(w, account)

	var wg sync.WaitGroup

	wg.Add(1)
	entries := make([]LEntry, 0, 20)
	go func() {
		defer wg.Done()

		var query string
		if permitted(w, r, user.ID, owner.ID) {
			query = `
SELECT e.id, e.private, e.body, e.created_at,
	   (SELECT COUNT(*) FROM comments c WHERE c.entry_id = e.id) as count
FROM entries e
WHERE e.user_id = ?
ORDER BY e.created_at DESC
LIMIT 20`
		} else {
			query = `
SELECT e.id, e.private, e.body, e.created_at,
	   (SELECT COUNT(*) FROM comments c WHERE c.entry_id = e.id) as count
FROM entries e
WHERE e.user_id = ? AND private = 0
ORDER BY e.created_at DESC
LIMIT 20`
		}

		rows, err := db.Query(query, owner.ID)
		if err != sql.ErrNoRows {
			checkErr(err)
		}
		for rows.Next() {
			var id, private, count int
			var body string
			var createdAt time.Time
			checkErr(rows.Scan(&id, &private, &body, &createdAt, &count))
			entry := LEntry{ID: id, Private: private == 1,
				Title:     strings.SplitN(body, "\n", 2)[0],
				Content:   strings.SplitN(body, "\n", 2)[1],
				CreatedAt: createdAt,
				Count:     count}
			entries = append(entries, entry)
		}
		rows.Close()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		markFootprint(user, owner.ID)
	}()

	wg.Wait()
	render(w, r, http.StatusOK, "entries.html", struct {
		Owner   *User
		Entries []LEntry
		Myself  bool
	}{owner, entries, user.ID == owner.ID})
}

type EComment struct {
	Comment     string
	NickName    string
	AccountName string
	CreatedAt   time.Time
}

type EOwner struct {
	id       int
	NickName string
}

func GetEntry(w http.ResponseWriter, r *http.Request) {
	user := authenticated(w, r)
	if user == nil {
		return
	}
	entryID := mux.Vars(r)["entry_id"]

	row := db.QueryRow(`
SELECT e.id, e.user_id, e.private, e.body, e.created_at, u.id, u.nick_name,
(CASE WHEN e.user_id = ? THEN 1
	  WHEN e.private = 1 AND NOT EXISTS (SELECT * FROM relations r WHERE r.one = e.user_id AND r.another = ?) THEN 0
	  ELSE 1 END) as permitted
FROM entries e
INNER JOIN users u ON u.id = e.user_id
WHERE e.id = ?`, user.ID, user.ID, entryID)
	var id, userID, private, permit int
	var body string
	var createdAt time.Time
	var owner EOwner
	err := row.Scan(&id, &userID, &private, &body, &createdAt, &owner.id, &owner.NickName, &permit)
	if err == sql.ErrNoRows {
		checkErr(ErrContentNotFound)
	}
	checkErr(err)
	if permit == 0 {
		checkErr(ErrPermissionDenied)
	}

	entry := Entry{id, userID, private == 1, strings.SplitN(body, "\n", 2)[0], strings.SplitN(body, "\n", 2)[1], createdAt}

	var wg sync.WaitGroup

	wg.Add(1)
	comments := make([]EComment, 0, 10)
	go func() {
		defer wg.Done()

		rows, err := db.Query(`
SELECT c.comment, c.created_at, u.nick_name, u.account_name
FROM comments c
INNER JOIN users u ON u.id = c.user_id
WHERE c.entry_id = ?`, entry.ID)
		if err != sql.ErrNoRows {
			checkErr(err)
		}
		for rows.Next() {
			c := EComment{}
			checkErr(rows.Scan(&c.Comment, &c.CreatedAt, &c.NickName, &c.AccountName))
			comments = append(comments, c)
		}
		rows.Close()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		markFootprint(user, owner.id)
	}()

	wg.Wait()

	render(w, r, http.StatusOK, "entry.html", struct {
		Owner    EOwner
		Entry    Entry
		Comments []EComment
	}{owner, entry, comments})
}

func PostEntry(w http.ResponseWriter, r *http.Request) {
	user := authenticated(w, r)
	if user == nil {
		return
	}

	title := r.FormValue("title")
	if title == "" {
		title = "タイトルなし"
	}
	content := r.FormValue("content")
	var private int
	if r.FormValue("private") == "" {
		private = 0
	} else {
		private = 1
	}
	_, err := db.Exec(`INSERT INTO entries (user_id, private, body) VALUES (?,?,?)`, user.ID, private, title+"\n"+content)
	checkErr(err)
	http.Redirect(w, r, "/diary/entries/"+user.AccountName, http.StatusSeeOther)
}

func PostComment(w http.ResponseWriter, r *http.Request) {
	user := authenticated(w, r)
	if user == nil {
		return
	}

	entryID := mux.Vars(r)["entry_id"]
	row := db.QueryRow(`SELECT * FROM entries WHERE id = ?`, entryID)
	var id, userID, private int
	var body string
	var createdAt time.Time
	err := row.Scan(&id, &userID, &private, &body, &createdAt)
	if err == sql.ErrNoRows {
		checkErr(ErrContentNotFound)
	}
	checkErr(err)

	entry := Entry{id, userID, private == 1, strings.SplitN(body, "\n", 2)[0], strings.SplitN(body, "\n", 2)[1], createdAt}
	owner := getUser(w, entry.UserID)
	if entry.Private {
		if !permitted(w, r, user.ID, owner.ID) {
			checkErr(ErrPermissionDenied)
		}
	}

	_, err = db.Exec(`INSERT INTO comments (entry_id, user_id, comment) VALUES (?,?,?)`, entry.ID, user.ID, r.FormValue("comment"))
	checkErr(err)
	http.Redirect(w, r, "/diary/entry/"+strconv.Itoa(entry.ID), http.StatusSeeOther)
}

type FFootprint struct {
	NickName    string
	AccountName string
	Updated     time.Time
}

func GetFootprints(w http.ResponseWriter, r *http.Request) {
	user := authenticated(w, r)
	if user == nil {
		return
	}

	footprints := make([]FFootprint, 0, 50)
	rows, err := db.Query(`
SELECT MAX(f.created_at) as updated, MIN(u.account_name), MIN(u.nick_name)
FROM footprints f
INNER JOIN users u ON u.id = f.owner_id
WHERE f.user_id = ?
GROUP BY f.user_id, f.owner_id, DATE(f.created_at)
ORDER BY updated DESC
LIMIT 50`, user.ID)
	if err != sql.ErrNoRows {
		checkErr(err)
	}
	for rows.Next() {
		fp := FFootprint{}
		checkErr(rows.Scan(&fp.Updated, &fp.AccountName, &fp.NickName))
		footprints = append(footprints, fp)
	}
	rows.Close()
	render(w, r, http.StatusOK, "footprints.html", struct{ Footprints []FFootprint }{footprints})
}

type FFriend struct {
	AccountName string
	NickName    string
	CreatedAt   time.Time
}

func GetFriends(w http.ResponseWriter, r *http.Request) {
	user := authenticated(w, r)
	if user == nil {
		return
	}
	rows, err := db.Query(`
SELECT r.created_at, u.account_name, u.nick_name
FROM relations r
INNER JOIN users u ON u.id = r.another
WHERE r.one = ?
ORDER BY r.created_at DESC`, user.ID)
	if err != sql.ErrNoRows {
		checkErr(err)
	}
	defer rows.Close()

	friends := make([]FFriend, 0, 100)
	for rows.Next() {
		var friend FFriend
		checkErr(rows.Scan(&friend.CreatedAt, &friend.AccountName, &friend.NickName))
		friends = append(friends, friend)
	}
	render(w, r, http.StatusOK, "friends.html", struct{ Friends []FFriend }{friends})
}

func PostFriends(w http.ResponseWriter, r *http.Request) {
	user := authenticated(w, r)
	if user == nil {
		return
	}

	anotherAccount := mux.Vars(r)["account_name"]
	another, isFriend := isFriendAccount(w, r, anotherAccount)
	if !isFriend {
		_, err := db.Exec(`INSERT INTO relations (one, another) VALUES (?,?), (?,?)`, user.ID, another.ID, another.ID, user.ID)
		checkErr(err)
		http.Redirect(w, r, "/friends", http.StatusSeeOther)
	}
}

func GetInitialize(w http.ResponseWriter, r *http.Request) {
	db.Exec("DELETE FROM relations WHERE id > 500000")
	db.Exec("DELETE FROM footprints WHERE id > 500000")
	db.Exec("DELETE FROM entries WHERE id > 500000")
	db.Exec("DELETE FROM comments WHERE id > 1500000")
}

func main() {
	runtime.GOMAXPROCS(4)

	var err error
	db, err = createDB()
	if err != nil {
		log.Fatalf("Failed to connect to DB: %s.", err.Error())
	}
	defer db.Close()

	// pool = newRedisPool("/tmp/redis.sock", 100)
	// defer pool.Close()

	ssecret := os.Getenv("ISUCON5_SESSION_SECRET")
	if ssecret == "" {
		ssecret = "beermoris"
	}
	store = sessions.NewCookieStore([]byte(ssecret))

	r := mux.NewRouter()

	l := r.Path("/login").Subrouter()
	l.Methods("GET").HandlerFunc(myHandler(GetLogin))
	l.Methods("POST").HandlerFunc(myHandler(PostLogin))
	r.Path("/logout").Methods("GET").HandlerFunc(myHandler(GetLogout))

	p := r.Path("/profile/{account_name}").Subrouter()
	p.Methods("GET").HandlerFunc(myHandler(GetProfile))
	p.Methods("POST").HandlerFunc(myHandler(PostProfile))

	d := r.PathPrefix("/diary").Subrouter()
	d.HandleFunc("/entries/{account_name}", myHandler(ListEntries)).Methods("GET")
	d.HandleFunc("/entry", myHandler(PostEntry)).Methods("POST")
	d.HandleFunc("/entry/{entry_id}", myHandler(GetEntry)).Methods("GET")

	d.HandleFunc("/comment/{entry_id}", myHandler(PostComment)).Methods("POST")

	r.HandleFunc("/footprints", myHandler(GetFootprints)).Methods("GET")

	r.HandleFunc("/friends", myHandler(GetFriends)).Methods("GET")
	r.HandleFunc("/friends/{account_name}", myHandler(PostFriends)).Methods("POST")

	r.HandleFunc("/initialize", myHandler(GetInitialize))
	r.HandleFunc("/", myHandler(GetIndex))
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("../static")))

	log.Fatal(unixSocketServe("/tmp/isucon_go.sock", r))
	// log.Fatal(http.ListenAndServe(":8080", r))
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
