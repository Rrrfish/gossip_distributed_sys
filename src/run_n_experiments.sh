#!/bin/bash

if [ "$#" -ne 4 ]; then
    echo "Usage: $0 [k] [size] [n] [output_file]"
    exit 1
fi

k=$1
size=$2
n=$3
output_file=$4
timeout_duration=5

echo "k,n,correct_result,gossip_result" > $output_file

for ((i=1; i<=n; i++))
do
    echo "Running experiment $i of $n with k=$k and size=$size"
    rm -f /var/tmp/distributedsys-hw-gossip-$(id -u)-*

    go run ./master.go $size > master_output.txt &
    master_pid=$!

    MASTERIDX=-1
    MASTER_SOCKET="/var/tmp/distributedsys-hw-gossip-$(id -u)-$MASTERIDX"
    while [ ! -e "$MASTER_SOCKET" ]; do
        sleep 0.1
    done

    declare -a worker_pids
    for (( idx=0; idx<$size; idx++ ))
    do
        WORKER_SOCKET="/var/tmp/distributedsys-hw-gossip-$(id -u)-$idx"
        if [ -e "$WORKER_SOCKET" ]; then
            rm "$WORKER_SOCKET"
        fi
        go run ./worker.go $k $size $idx > worker_output_$idx.txt &
        worker_pids[$idx]=$!
    done

    (
        sleep $timeout_duration
        echo "Experiment $i timed out after $timeout_duration seconds."
        kill $master_pid
        for pid in "${worker_pids[@]}"
        do
            kill $pid 2>/dev/null
        done
        echo "timeout" > experiment_timeout
    ) &

    timer_pid=$!

    wait $master_pid

    if [ -e "experiment_timeout" ]; then
        rm experiment_timeout
        wait $timer_pid 2>/dev/null
        rm master_output.txt
        rm worker_output_*.txt
        rm -f /var/tmp/distributedsys-hw-gossip-$(id -u)-*
        echo "$k,$i,timeout,timeout" >> $output_file
        continue
    else
        kill $timer_pid 2>/dev/null
    fi

    for pid in "${worker_pids[@]}"
    do
        wait $pid 2>/dev/null
    done

    correct_result=$(grep "\[master\] correct average wage is:" master_output.txt | awk '{print $6}')
    gossip_result=$(grep "gossip result is:" worker_output_*.txt | awk '{print $4}' | head -n 1)

    echo "$k,$i,$correct_result,$gossip_result" >> $output_file

    rm master_output.txt
    rm worker_output_*.txt
    rm -f /var/tmp/distributedsys-hw-gossip-$(id -u)-*
done
