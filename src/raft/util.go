package raft

import (
	"log"
	"math/rand"
	"time"
)

// Debugging
const Debug = true

func DPrintf(format string, a ...interface{}) {
	if Debug {
		log.Printf(format, a...)
	}
}

func (rf *Raft) lastLogTerm() int {
	if len(rf.logs) == 0 {
		return 0
	}
	return rf.logs[len(rf.logs)-1].Term
}
func (rf *Raft) lastLogIndex() int {
	return len(rf.logs)
}

// 随机时间
func randTime(min int64, max int64) time.Duration {
	dura := rand.Int63()%(max-min) + min
	return time.Millisecond * time.Duration(dura)
}

func RandElectionTime() time.Duration {
	return randTime(150, 500)
}
func RandHeartBeatTime() time.Duration {
	return randTime(50, 150)
}
