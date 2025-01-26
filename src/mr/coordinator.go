package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"
)

type TaskItem struct {
	TaskID    int
	Files     []string
	Completed bool
}

type Coordinator struct {
	// Your definitions here.
	mu          sync.Mutex
	taskID      int               // 任务ID
	nReduce     int               //reduce分块数量
	mapfiles    chan string       // 文件队列
	reducefiles []chan string     // 文件队列
	tasks       map[int]*TaskItem // 实时任务表
	// tasks       chan *TaskItem // 任务队列
}

// Your code here -- RPC handlers for the worker to call.

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// 分配任务
// 还有map任务未完成，不能分配reduce任务
func (c *Coordinator) DispatchTask(args *TaskArgs, reply *TaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	reply.NumReduce = c.nReduce
	// 给任务分配文件
	var files []string
	var ch chan string
	var mytask *TaskItem
	reply.TaskID = c.taskID
	c.taskID++
	if len(c.mapfiles) > 0 {
		// 分配map任务
		reply.TaskType = MapTask
		ch = c.mapfiles
	} else if len(c.tasks) > 0 {
		// 等待任务完成
		reply.TaskType = WaitTask
		return nil
	} else if idx := c.getReduceTask(); idx != -1 {
		// 分配reduce任务
		reply.TaskType = ReduceTask
		ch = c.reducefiles[idx]
	} else {
		// 下班
		reply.TaskType = DoneTask
		return nil
	}
	// map只分配一个文件
	if reply.TaskType == MapTask {
		files = append(files, <-ch)
	} else {
		for len(ch) > 0 {
			files = append(files, <-ch)
		}
	}
	reply.InputFiles = files
	mytask = &TaskItem{TaskID: reply.TaskID, Files: files, Completed: false}
	c.tasks[mytask.TaskID] = mytask // 记录任务
	// 超时处理
	time.AfterFunc(10*time.Second, func() {
		if mytask.Completed {
			return
		}
		log.Printf("timeout: %d\n", mytask.TaskID)
		c.mu.Lock()
		defer c.mu.Unlock()
		// 超时，重新分配任务
		for _, file := range mytask.Files {
			c.mapfiles <- file
		}
		delete(c.tasks, mytask.TaskID)
	})
	log.Printf("Dispatch: id %d, type %d, files %v\n", reply.TaskID, reply.TaskType, files)
	return nil
}

func (c *Coordinator) getReduceTask() int {
	for i, ch := range c.reducefiles {
		if len(ch) > 0 {
			return i
		}
	}
	return -1
}

// FinishTask 任务完成
func (c *Coordinator) FinishTask(args *FinishArgs, reply *TaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	log.Printf("Finish: type %d, id %d\n", args.TaskType, args.TaskID)
	task := c.tasks[args.TaskID]
	if task == nil {
		return fmt.Errorf("task %d not found", args.TaskID)
	}
	makeReduce := func(filename string, mapid int) {
		pattern := regexp.MustCompile(`^mr-(\d+)-(\d+)$`)
		matches := pattern.FindStringSubmatch(filename)
		if len(matches) == 3 {
			mid, _ := strconv.Atoi(matches[1])
			if mid != -1 && mid != mapid {
				return
			}
			rid, _ := strconv.Atoi(matches[2])
			if rid < c.nReduce {
				c.reducefiles[rid] <- filename
			}
		}
	}

	if args.Err != nil {
		// 任务失败，重新添加文件
		for _, file := range task.Files {
			if args.TaskType == MapTask {
				c.mapfiles <- file
			} else if args.TaskType == ReduceTask {
				makeReduce(file, -1)
			}
		}
		return nil
	}
	task.Completed = true
	delete(c.tasks, args.TaskID) // 删除任务，之后重新分配
	// 完成map任务，创建reduce任务需要的文件
	if args.TaskType == MapTask {
		dir, _ := os.Getwd()
		// 获取符合条件的文件名，用作reduce任务的输入
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				makeReduce(info.Name(), args.TaskID)
			}
			return nil
		})
	}
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	// 所有reduce任务完成
	reduced := true
	for _, ch := range c.reducefiles {
		if len(ch) > 0 {
			reduced = false
			break
		}
	}
	// Your code here.
	// 没有待处理文件 且 没有任务在运行
	return len(c.mapfiles) == 0 && reduced && len(c.tasks) == 0
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}
	log.Printf("coordinator: reduce %d, files %d\n", nReduce, len(files))
	// Your code here.
	c.mu = sync.Mutex{}
	c.nReduce = nReduce
	c.mapfiles = make(chan string, len(files))
	// reduce任务分桶处理
	c.reducefiles = make([]chan string, nReduce)
	for i := 0; i < nReduce; i++ {
		c.reducefiles[i] = make(chan string, len(files))
	}
	c.tasks = make(map[int]*TaskItem)
	for _, file := range files {
		c.mapfiles <- file
	}
	c.server()
	return &c
}
