#!/bin/sh

./goto -rpc=true -host=localhost:8080 &
sleep 1
./goto -master=localhost:8080 -http=:8081 &
./goto -master=localhost:8080 -http=:8082 &
./goto -master=localhost:8080 -http=:8083 &
