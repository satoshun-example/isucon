#!/bin/sh

cd webapp/

GOARCH=amd64 GOOS=linux go build -v -o golang-webapp .
scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
    -P 2222 \
    -i /Users/satouhayabusa/git/github.com/satoshun-example/isucon/4/.vagrant/machines/default/virtualbox/private_key \
    golang-webapp vagrant@127.0.0.1:

ssh -p 2222 \
    -i /Users/satouhayabusa/git/github.com/satoshun-example/isucon/4/.vagrant/machines/default/virtualbox/private_key \
    -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
    vagrant@127.0.0.1 \
    "sudo mv golang-webapp /home/isucon/"
