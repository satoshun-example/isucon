package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/gorilla/sessions"
)

func getEnv(key string, def string) string {
	v := os.Getenv(key)
	if len(v) == 0 {
		return def
	}

	return v
}

func getFlash(session *sessions.Session, key string) string {
	if value, ok := session.Values[key]; ok {
		delete(session.Values, key)
		return value.(string)
	}

	return ""
}

func calcPassHash(password, hash string) string {
	h := sha256.New()
	io.WriteString(h, password)
	io.WriteString(h, ":")
	io.WriteString(h, hash)

	return fmt.Sprintf("%x", h.Sum(nil))
}
