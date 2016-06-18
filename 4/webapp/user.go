package main

import (
	"time"
)

type User struct {
	ID           int
	Login        string
	PasswordHash string
	Salt         string
}

type LastLogin struct {
	Login     string
	IP        string
	CreatedAt time.Time
}

func getLastLogin(userID interface{}) *LastLogin {
	rows, err := db.Query(
		"SELECT login, ip, created_at FROM login_log WHERE succeeded = 1 AND user_id = ? ORDER BY id DESC LIMIT 2",
		userID,
	)

	if err != nil {
		return nil
	}
	defer rows.Close()

	lastLogin := new(LastLogin)
	for rows.Next() {
		err = rows.Scan(&lastLogin.Login, &lastLogin.IP, &lastLogin.CreatedAt)
		if err != nil {
			lastLogin = nil
			return nil
		}
	}

	return lastLogin
}
