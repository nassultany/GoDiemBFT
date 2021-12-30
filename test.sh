#!/bin/sh

trap "trap - SIGTERM && kill -- -$$" SIGINT SIGTERM EXIT

echo "The script pid is $$"

# The sleeps are to ensure that the validators get spawned in the order specified below
./diem -master true &
sleep .2
./diem -grpcPort 8091 -port 9091 -clusterAddr 127.0.0.1:8000 &
sleep .1
./diem -grpcPort 8092 -port 9092 -clusterAddr 127.0.0.1:8000 &
sleep .1
./diem -grpcPort 8093 -port 9093 -clusterAddr 127.0.0.1:8000 &

#wait
sleep 7

kill -- -$$