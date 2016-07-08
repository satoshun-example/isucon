#!/bin/bash

# go get -u -v github.com/go-sql-driver/mysql
# go get -u -v github.com/gorilla/sessions
GOARCH=amd64 GOOS=linux go build -v -o golang-webapp
vagrant ssh -c "sudo supervisorctl restart isucon_go"
