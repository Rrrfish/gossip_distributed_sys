package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"

	"gossip/myrpc"
)

// 每一个Worker在被生成后向Master询问自己的具体薪资（领工资？），Master会给Worker一个随机数

// 一开始设想
// 为了加快执行速度，系统的正式开始Gossip的阶段不需要所有Worker都确认完薪资，每个确认好的Wroker只需将state设置为Active即可
// 但是想到因为可能会存在处理速度过快，Worker没有全部确认完薪资之后前面那些Worker就静默（Removed）了！所以就不采取了

//
// Worker
//

const tolerance = 1e-6
const checkDuration = 200

type Worker struct {
	idx          int
	wage         float64 // 计算所有Worker该字段的平均数
	state        State   // 是否准备好参与平均数计算
	mu           sync.Mutex
	k            int
	sockets      []int // 储存还在活跃的Worker的idx（也就是Socket）
	size         int
	masterSocket string
	exchanging   bool // 表示是否正在进行工资交换
}

type State int

const (
	Idle   State = iota // 还没领工资
	Ready               // 准备阶段
	Active              // gossip阶段
	Silent              // gossip结束，静默
)

func MakeWorker(k int, size int, id int) *Worker {
	socket := make([]int, size)
	for i := 0; i < size; i++ {
		socket[i] = i
	}
	return &Worker{idx: id, k: k, state: Idle, sockets: socket, size: size, masterSocket: myrpc.GetSock(myrpc.Masteridx)}
}

func (w *Worker) start() {

	for {
		w.mu.Lock()
		if w.state == Ready {
			// fmt.Printf("worker %v receives wage: %v\n", w.idx, w.wage)
			w.mu.Unlock()
			break
		}
		w.mu.Unlock()
		w.receiveWage()
		// fmt.Println("receiveWage refuse: worker ", w.idx)
		time.Sleep(5 * time.Millisecond)
	}

	w.notifyMasterReady()

	fmt.Println("worker starts: ", w.idx)

	w.mu.Lock()
	for w.state != Active {
		w.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		w.mu.Lock()
	}

	w.mu.Unlock()

	// fmt.Printf("!!!!!!!!%v can gossip now, it is Active\n", w.idx)

	go func() {
		for {
			w.mu.Lock()

			if w.state == Silent {
				w.mu.Unlock()
				return
			}
			w.mu.Unlock()
			w.whatsYourWage()
			time.Sleep(10 * time.Millisecond)
		}
	}()

	for {
		w.mu.Lock()

		if w.state == Silent {
			w.mu.Unlock()
			return
		}
		w.mu.Unlock()

		time.Sleep(checkDuration * time.Millisecond)
	}
}

func (w *Worker) receiveWage() {
	w.mu.Lock()
	defer w.mu.Unlock()
	args := myrpc.ReceiveWageArgs{}
	reply := myrpc.ReceiveWageReply{}

	ok := w.callMaster("Master.RecruitWorkers", &args, &reply)
	if !ok {
		log.Fatal("calling RPC Master.RecruitWorkers errs")
	}

	w.wage = reply.Wage
	w.state = Ready
	// fmt.Printf("No.%v receiveWage %v!!\n", w.idx, w.wage)
}

func (w *Worker) whatsYourWage() {
	w.mu.Lock()
	if w.exchanging {
		w.mu.Unlock()
		return
	}
	if w.size == 1 {
		w.state = Silent
		w.mu.Unlock()
		return
	}
	w.exchanging = true
	currentWage := w.wage
	w.mu.Unlock()

	// fmt.Printf("No.%v: what's your wage? my wage is %v\n", w.idx, w.wage)
	args := myrpc.WhatsYourWageArgs{Wage: currentWage}
	reply := myrpc.WhatsYourWageReply{}

	ok := w.callWorker("Worker.Snooped", &args, &reply)
	if !ok {
		w.mu.Lock()
		w.exchanging = false
		w.mu.Unlock()
		// fmt.Println("calling RPC Worker.Snooped errs, it is busy")
		return
	}

	if reply.Busy {
		w.mu.Lock()
		w.exchanging = false
		w.mu.Unlock()
		// time.Sleep(10 * time.Millisecond)
		return
	}

	if reply.Slient {
		w.mu.Lock()
		w.exchanging = false
		w.remove(reply.Idx)
		w.mu.Unlock()
		return
	}

	var needToLeave bool
	w.mu.Lock()
	w.exchanging = false

	if math.Abs(w.wage-reply.Wage) < tolerance && w.bored() {
		needToLeave = true
	} else {
		w.wage = (w.wage + reply.Wage) / 2
		// fmt.Printf("%v: my new wage is %v\n", w.idx, w.wage)
	}
	w.mu.Unlock()

	if needToLeave {
		w.leave()
	}
}

func (w *Worker) StartGossip(args *myrpc.StartGossipArgs, reply *myrpc.StartGossipReply) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.state = Active
	return nil
}

func (w *Worker) Snooped(args *myrpc.WhatsYourWageArgs, reply *myrpc.WhatsYourWageReply) error {
	w.mu.Lock()
	reply.Idx = w.idx
	if w.state == Silent {
		w.mu.Unlock()
		reply.Slient = true
		return nil
	}
	if w.exchanging {
		w.mu.Unlock()
		reply.Busy = true
		return nil
	}
	w.exchanging = true
	currentWage := w.wage
	w.mu.Unlock()

	reply.Wage = currentWage
	reply.Busy = false
	w.mu.Lock()

	var needToLeave bool
	if math.Abs(w.wage-args.Wage) < tolerance && w.bored() {
		needToLeave = true
	} else {
		w.wage = (w.wage + args.Wage) / 2
		// fmt.Printf("[%v says]: before: her wage %v my wage %v, Now my wage is %v\n", w.idx, args.Wage, currentWage, w.wage)

	}
	w.exchanging = false
	w.mu.Unlock()

	if needToLeave {
		w.leave()
	}
	return nil
}

func (w *Worker) leave() {
	w.mu.Lock()
	w.state = Silent
	w.exchanging = true
	w.mu.Unlock()

	args := myrpc.ShrinkArgs{Idx: w.idx}
	reply := myrpc.ShrinkReply{}
	fmt.Printf("%v 我走了！我的工资：%v\n", w.idx, w.wage)
	for _, sock := range w.sockets {
		if sock != w.idx {
			myrpc.Call("Worker.Shrink", myrpc.GetSock(sock), &args, &reply)
		}
	}
	// w.callWorker("Worker.Shrink", &args, &reply)
}

func (w *Worker) bored() bool {
	state := false
	// fmt.Printf("k为: %v\n", w.k)
	if rand.Float64() < 1.0/float64(w.k) {
		state = true
	}
	return state
}

func (w *Worker) randomFetchNodeSock() string {
	// fmt.Println("randomFetchNodeSock")
	w.mu.Lock()
	size := w.size
	// w.mu.Unlock()

	randIdx := w.idx
	for randIdx == w.idx {
		// fmt.Println("?")
		randIdx = w.sockets[rand.Intn(size)] // [0, w.size)
	}
	// w.mu.Lock()
	sockname := myrpc.GetSock(randIdx)
	// sockname := myrpc.GetSock(randIdx)
	if size < 5 {

		fmt.Printf("%v, 目前有%v个worker,获取到socket: %v\n", w.idx, size, sockname)
	}
	w.mu.Unlock()
	// fmt.Println("sockname: ", sockname)

	return sockname
}

// 减少sock slice里的对应worker
func (w *Worker) Shrink(args *myrpc.ShrinkArgs, reply *myrpc.ShrinkReply) error {
	idx := args.Idx

	w.mu.Lock()
	w.remove(idx)
	w.mu.Unlock()
	return nil
}

func (w *Worker) remove(idx int) {
	for i, val := range w.sockets {
		if val == idx {
			w.sockets = append(w.sockets[:i], w.sockets[i+1:]...)
			// fmt.Printf("%v receives shrink, size now is %v\n", w.idx, w.size)
			w.size--
			break
		}
	}
}

func (w *Worker) notifyMasterReady() {
	args := myrpc.WorkerReadyArgs{WorkerID: w.idx}
	reply := myrpc.WorkerReadyReply{}
	// fmt.Printf("%v 我准备好了！\n", w.idx)
	ok := w.callMaster("Master.WorkerReady", &args, &reply)
	if !ok {
		log.Fatal("Worker failed to notify master it is ready")
	}
}

// RPC调用Worker的方法
func (w *Worker) callWorker(rpcname string, args interface{}, reply interface{}) bool {
	// fmt.Println("callWorker")
	// sockname := w.randomFetchNodeSock()
	// fmt.Println("sockname is : ", sockname)
	// fmt.Println("callWorker")
	// return myrpc.Call(rpcname, sockname, args, reply)
	var sockname string
	var ok bool
	for retries := 0; retries < 5; retries++ {
		sockname = w.randomFetchNodeSock()
		ok = myrpc.Call(rpcname, sockname, args, reply)
		if ok {
			return true
		} else {
			time.Sleep(50 * time.Millisecond)
			continue
		}
	}
	return false
}

func (w *Worker) callMaster(rpcname string, args interface{}, reply interface{}) bool {
	ok := myrpc.Call(rpcname, w.masterSocket, args, reply)
	if !ok {
		return false
	}
	return true
}

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Usage: worker [k] [size] [id]")
		os.Exit(1)
	}
	k, err1 := strconv.Atoi(os.Args[1])
	size, err2 := strconv.Atoi(os.Args[2])
	id, err3 := strconv.Atoi(os.Args[3])
	if err1 != nil || err2 != nil || err3 != nil {
		log.Fatal("can't convert to integer.")
	}
	sock := myrpc.GetSock(id)

	worker := MakeWorker(k, size, id)

	rpc.Register(worker)
	rpc.HandleHTTP()
	os.Remove(sock)
	l, e := net.Listen("unix", sock)
	if e != nil {
		log.Fatal("listen err: ", e)
	}

	go http.Serve(l, nil)

	// worker.notifyMasterReady()

	worker.start()
	worker.mu.Lock()
	if worker.size == 1 {
		fmt.Println("gossip result is: ", worker.wage)
	}
	fmt.Printf("%v结束 还剩下%v个worker\n", worker.idx, worker.size)
	worker.mu.Unlock()
}
