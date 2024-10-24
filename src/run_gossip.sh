#!/bin/bash

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 [k] [size]"
    exit 1
fi

k=$1
size=$2

# 获取 master 的 socket 路径
MASTERIDX=-1
MASTER_SOCKET="/var/tmp/distributedsys-hw-gossip-$(id -u)-$MASTERIDX"

# 删除旧的 socket 文件（如果存在）
if [ -e "$MASTER_SOCKET" ]; then
    rm "$MASTER_SOCKET"
fi

# 启动 master 程序
go run ./master.go $size &

# 等待 master 的 socket 文件被创建，确保 master 已经启动并准备好接受 RPC 请求
while [ ! -e "$MASTER_SOCKET" ]; do
    sleep 0.1
done

# 并发启动 size 个 worker 程序
for (( idx=0; idx<$size; idx++ ))
do
    WORKER_SOCKET="/var/tmp/distributedsys-hw-gossip-$(id -u)-$idx"
    if [ -e "$WORKER_SOCKET" ]; then
        rm "$WORKER_SOCKET"
    fi
    go run ./worker.go $k $size $idx &
done

# 等待所有 worker 的 socket 文件被创建
for (( idx=0; idx<$size; idx++ ))
do
    WORKER_SOCKET="/var/tmp/distributedsys-hw-gossip-$(id -u)-$idx"
    while [ ! -e "$WORKER_SOCKET" ]; do
        sleep 0.1
    done
done

wait
