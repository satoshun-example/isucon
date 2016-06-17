#!/bin/bash

go get -u -v github.com/go-martini/martini
go get -u -v github.com/go-sql-driver/mysql
go get -u -v github.com/martini-contrib/render
go get -u -v github.com/martini-contrib/sessions
go build -o golang-webapp .
