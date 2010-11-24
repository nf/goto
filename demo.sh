#!/bin/sh

echo "Starting up"
cd ../stat/server
./stats &
stats_pid=$!
cd ../../goto
sleep 1
./goto -rpc=true -host=localhost:8080 &
master_pid=$!
sleep 1
./goto -master=localhost:8080 -http=:8081 &
slave1_pid=$!
./goto -master=localhost:8080 -http=:8082 &
slave2_pid=$!
./goto -master=localhost:8080 -http=:8083 &
slave3_pid=$!
sleep 1

echo "Testing the master (n=1)"
bench/bench -host=localhost:8080 -n=1 &
pid=$!
sleep 15
kill $pid

echo "Testing the master (n=10)"
bench/bench -host=localhost:8080 -n=10 &
pid=$!
sleep 15
kill $pid

echo "Testing 1 slave (n=10)"
bench/bench -host=localhost:8081 -n=10 &
pid=$!
sleep 15
kill $pid

echo "Testing 2 slaves (n=10)"
bench/bench -host=localhost:8081,localhost:8082 -n=10 &
pid=$!
sleep 15
kill $pid

echo "Testing 3 slaves (n=10)"
bench/bench -host=localhost:8081,localhost:8082,localhost:8083 -n=10 &
pid=$!
sleep 15
kill $pid

echo "Testing 3 slaves (n=20)"
bench/bench -host=localhost:8081,localhost:8082,localhost:8083 -n=20 &
pid=$!
sleep 30 
kill $pid

echo "Shutting down"
kill $stats_pid
kill $master_pid
kill $slave1_pid
kill $slave2_pid
kill $slave3_pid
echo "Done"
