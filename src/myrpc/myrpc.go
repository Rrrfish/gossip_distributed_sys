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

//
// 创建一个网络
//

// const masteridx = -1

// var con config

// type config struct {
// 	sockets      map[int]sock
// 	size         int
// 	masterSocket sock
// 	mu           sync.Mutex
// 	k            int
// 	result       float64
// 	workers      []*Worker
// }

// type sock string

// type WorkerRPCConfig interface {
// 	test()
// 	randomFetchNodeSock(idx int) sock
// 	removeWorker(idx int, wage float64)
// 	call(rpcname string, sockname sock, args interface{}, reply interface{}) bool
// }

// func SetConfig(k int, n int) config {
// 	// rpc.Register(new(Master))
// 	// rpc.Register(new(Worker))
// 	rpc.HandleHTTP()
// 	return config{
// 		sockets: make(map[int]sock, n),
// 		size:    0,
// 		workers: make([]*Worker, 0),
// 		k:       k,
// 	}
// }

// func (c *config) test() {
// 	fmt.Println("test")
// }

// func (c *config) startGossip() {
// 	fmt.Println("start!!!!")

// 	// c.mu.Lock()
// 	// defer c.mu.Unlock()
// 	len := c.size
// 	for i := 0; i < len; i++ {
// 		go c.workers[i].start()
// 	}

// 	fmt.Println("^^^")
// }

// //
// // Master
// //

// // 注册master，生成socket，便于后续进行RPC通信
// func (c *config) registerMaster(m *Master) {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()
// 	rpc.Register(m)
// 	sockname := getSock(-1) // 记为-1
// 	server(sockname)
// }

// //
// // Worker
// //

// // 随机选一个Worker的Socket
// func (c *config) randomFetchNodeSock(idx int) sock {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()
// 	randidx := idx
// 	for randidx == idx {
// 		randidx = int(rand.Int63()) % (c.size)
// 	}

// 	sockname := getSock(randidx)
// 	return sockname
// }

// // 注册worker，生成socket，便于后续进行RPC通信
// func (c *config) registerWorker(worker *Worker) {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()
// 	c.workers = append(c.workers, worker)
// 	worker.idx = c.size
// 	worker.k = c.k
// 	fmt.Println("worker k is ", worker.k)
// 	c.size++
// 	fmt.Printf("config has %v workers\n", c.size)
// 	sockname := getSock(worker.idx)
// 	c.sockets[worker.idx] = sockname
// 	fmt.Printf("%v has sockname %v\n", worker.idx, sockname)

// 	rpc.Register(worker)
// 	rpc.HandleHTTP()
// 	server(sockname)
// 	// go worker.start()
// }

// func (c *config) removeWorker(idx int, wage float64) {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()
// 	delete(c.sockets, idx)
// 	c.size--
// 	if c.size == 0 {
// 		c.result = wage
// 	}
// }

// // RPC调用Worker的方法
// func (c *config) call(rpcname string, sockname sock, args interface{}, reply interface{}) bool {

// 	con, err := rpc.DialHTTP("unix", string(sockname))
// 	if err != nil {
// 		log.Fatal("dialing:", err)
// 	}
// 	defer con.Close()

// 	err = con.Call(rpcname, args, reply)
// 	if err == nil {
// 		return true
// 	}

// 	fmt.Println(err)
// 	return false
// }

// // 开一个线程listen RPC请求
// func server(sockname sock) {
// 	// 防止上次生成的套接字文件未清理，先清一下
// 	os.Remove(string(sockname))
// 	l, e := net.Listen("unix", string(sockname))
// 	if e != nil {
// 		log.Fatal("listen error:", e)
// 	}
// 	go http.Serve(l, nil)
// }

// func (c *config) Done() bool {
// 	ret := false
// 	if c.size == 0 {
// 		ret = true
// 	}
// 	return ret
// }

// // 该函数取自MIT6.5840 Lab1 MapReduce中的框架代码，个人感觉比较好用，改了一下变量名。
// // 原注释：
// // Cook up a unique-ish UNIX-domain socket name
// // in /var/tmp, for the coordinator.
// // Can't use the current directory since
// // Athena AFS doesn't support UNIX-domain sockets.
// func getSock(nodeidx int) sock {
// 	nodename := strconv.Itoa(nodeidx)
// 	s := "/var/tmp/distributedsys-hw-gossip-" // 文件名称改了一下
// 	s += strconv.Itoa(os.Getuid())
// 	s += "-" + nodename
// 	return sock(s)
// }

// func (c *config) ShowResult() float64 {
// 	return c.result
// }

// // Role: Master & Worker
// // Master 用传统方式计算平均数，便于后续计算Gossip结果的误差
// // Worker Gossip中的每一个节点，这里用多线程模拟分布式环境

// // 各个节点间利用RPC通信
// // myrpc.go创造一个网络环境

// func main() {
// 	if len(os.Args) < 3 {
// 		fmt.Fprintf(os.Stderr, "need k and the number of nodes\n")
// 		os.Exit(1)
// 	}

// 	k, err1 := strconv.Atoi(os.Args[1])

// 	n, err2 := strconv.Atoi(os.Args[2])
// 	if err1 != nil || err2 != nil {
// 		fmt.Fprintf(os.Stderr, "not a number\n")
// 		os.Exit(1)
// 	}

// 	c := SetConfig(k, n)
// 	fmt.Println("?")

// 	master := MakeMaster()
// 	c.registerMaster(&master)
// 	fmt.Println("??")
// 	for i := 0; i < n; i++ {
// 		worker := MakeWorker(WorkerRPCConfig(&c), k)
// 		fmt.Println("k:", worker.k)
// 		c.registerWorker(&worker)
// 		// go worker.start()
// 		worker.receiveWage()
// 	}

// 	c.startGossip()

// 	if !c.Done() {
// 		fmt.Println("a?")
// 		time.Sleep(200 * time.Millisecond)
// 	}

// 	gossipRes := c.ShowResult()
// 	traditionalRes := master.calMean()
// 	fmt.Println(master.sum, master.workerNum)
// 	fmt.Printf("gossip result: %v, correct result: %v\n", gossipRes, traditionalRes)
// }
