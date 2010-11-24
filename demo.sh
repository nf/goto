#!/bin/sh

STATS=127.0.0.1:8090
MASTER=127.0.0.1:8080
N1=1
N2=8
N3=16

echo "Starting up"
cd ../stat/server
./stats &
stats_pid=$!
cd ../../goto
sleep 1
./goto -stats=$STATS -host=$MASTER -rpc=true &
master_pid=$!
sleep 1
./goto -stats=$STATS -host=$MASTER -master=$MASTER -http=:8081 &
slave1_pid=$!
./goto -stats=$STATS -host=$MASTER -master=$MASTER -http=:8082 &
slave2_pid=$!
./goto -stats=$STATS -host=$MASTER -master=$MASTER -http=:8083 &
slave3_pid=$!
sleep 1

echo "Testing the master (n=$N1)"
bench/bench -stats=$STATS -host=$MASTER -n=$N1 &
pid=$!
read
kill $pid

echo "Testing the master (n=$N2)"
bench/bench -stats=$STATS -host=$MASTER -n=$N2 &
pid=$!
read
kill $pid

echo "Testing 1 slave (n=$N2)"
bench/bench -stats=$STATS -host=127.0.0.1:8081 -n=$N2 &
pid=$!
read
kill $pid

echo "Testing 2 slaves (n=$N2)"
bench/bench -stats=$STATS -host=127.0.0.1:8081,127.0.0.1:8082 -n=$N2 &
pid=$!
read
kill $pid

echo "Testing 3 slaves (n=$N2)"
bench/bench -stats=$STATS -host=127.0.0.1:8081,127.0.0.1:8082,127.0.0.1:8083 -n=$N2 &
pid=$!
read
kill $pid

echo "Testing 3 slaves (n=$N3)"
bench/bench -stats=$STATS -host=127.0.0.1:8081,127.0.0.1:8082,127.0.0.1:8083 -n=$N3 &
pid=$!
read
kill $pid

echo "Shutting down"
kill $stats_pid
kill $master_pid
kill $slave1_pid
kill $slave2_pid
kill $slave3_pid
