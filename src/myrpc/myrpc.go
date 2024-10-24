package myrpc

import (
	"fmt"
	"net/rpc"
	"os"
	"strconv"
)

const Masteridx = -1

type WorkerReadyArgs struct {
	WorkerID int
}

type WorkerReadyReply struct{}

type ReceiveWageArgs struct{}

type ReceiveWageReply struct {
	Wage float64
}

type WhatsYourWageArgs struct {
	Wage float64
}

type WhatsYourWageReply struct {
	Wage   float64
	Busy   bool
	Slient bool
	Idx    int
}

type StartGossipArgs struct {
}

type StartGossipReply struct {
}

type ShrinkArgs struct {
	Idx int
}

type ShrinkReply struct {
}

// RPC调用Worker的方法
func Call(rpcname string, sockname string, args interface{}, reply interface{}) bool {
	con, err := rpc.DialHTTP("unix", string(sockname))
	if err != nil {
		// log.Fatal("dialing:", err)
		// fmt.Println("dialing: ", err)
		return false
	}
	defer con.Close()

	err = con.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}

// 该函数取自MIT6.5840 Lab1 MapReduce中的框架代码，个人感觉比较好用，改了一下变量名。
// 原注释：
// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func GetSock(nodeidx int) string {
	nodename := strconv.Itoa(nodeidx)
	s := "/var/tmp/distributedsys-hw-gossip-" // 文件名称改了一下
	s += strconv.Itoa(os.Getuid())
	s += "-" + nodename
	return s
}
