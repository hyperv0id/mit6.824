package mr

import (
	"os"
	"strconv"
)

//
// RPC definitions.
//
// remember to capitalize all names.
//

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

// Add your RPC definitions here.

type TaskType int

const (
	MapTask TaskType = iota
	ReduceTask
	WaitTask
	DoneTask
)

type TaskArgs struct {
	WorkerID int
}

type TaskReply struct {
	TaskID     int
	TaskType   TaskType
	NumReduce  int
	InputFiles []string
}

type FinishArgs struct {
	TaskID   int
	TaskType TaskType
	Err      error
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
