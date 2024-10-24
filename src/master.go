package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"sync"

	"gossip/myrpc"
)

//
// Master
//

type Master struct {
	// averageWage float64 // 传统方法计算的平均数，方便后续计算误差
	sum          float64
	workerSum    int
	workerNum    int
	mu           sync.Mutex
	readyWorkers int
	readyCond    *sync.Cond
}

func MakeMaster(workerSum int) *Master {
	m := &Master{
		workerSum: workerSum,
	}
	m.readyCond = sync.NewCond(&m.mu)
	return m
}

// 确认工资（发工资？）,2-10（单位为K）
func assignWage() float64 {
	wage := rand.Int63()%8 + 2
	return float64(wage)
}

// 给Worker发工资并记录工资
func (m *Master) RecruitWorkers(args *myrpc.ReceiveWageArgs, reply *myrpc.ReceiveWageReply) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	reply.Wage = assignWage()
	m.workerNum++
	m.sum += reply.Wage
	// fmt.Printf("现在master共计: %vk, %v个Worker\n", m.sum, m.workerNum)
	return nil
}

func (m *Master) WorkerReady(args *myrpc.WorkerReadyArgs, reply *myrpc.WorkerReadyReply) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readyWorkers++
	// fmt.Printf("Worker %d is ready (%d/%d)\n", args.WorkerID, m.readyWorkers, m.workerSum)
	if m.readyWorkers == m.workerSum {
		m.readyCond.Broadcast()
	}
	return nil
}

func (m *Master) startWorkerGossip() {
	for i := range m.workerNum {
		go myrpc.Call("Worker.StartGossip", myrpc.GetSock(i), &myrpc.StartGossipArgs{}, &myrpc.StartGossipReply{})
	}
}

// 计算平均数
func (m *Master) calMean() float64 {
	fmt.Println("m.workerNum is ", m.workerNum)
	if m.workerNum == 0 {
		log.Fatal("can not devide by 0")
	}
	return m.sum / float64(m.workerNum)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: worker [workersum]")
		os.Exit(1)
	}
	sum, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatal("[master] can't convert to integer.")
	}
	sock := myrpc.GetSock(myrpc.Masteridx)

	master := MakeMaster(sum)

	rpc.Register(master)

	rpc.HandleHTTP()
	os.Remove(sock)
	l, e := net.Listen("unix", sock)
	if e != nil {
		log.Fatal("listen err: ", e)
	}

	go http.Serve(l, nil)

	master.mu.Lock()
	for master.readyWorkers < master.workerSum {
		master.readyCond.Wait()
	}
	master.mu.Unlock()
	// fmt.Println("master够了")

	for idx := 0; idx < master.workerSum; idx++ {
		func() {
			sockname := myrpc.GetSock(idx)
			args := myrpc.StartGossipArgs{}
			reply := myrpc.StartGossipReply{}
			ok := myrpc.Call("Worker.StartGossip", sockname, &args, &reply)
			if !ok {
				fmt.Printf("Failed to send start signal to worker %d\n", idx)
			}
		}()
	}

	fmt.Println("[master] corret average num is: ", master.calMean())
}
