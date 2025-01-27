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
	TaskType  TaskType
}

type Coordinator struct {
	// Your definitions here.
	mu          sync.Mutex
	taskID      int               // 任务ID
	nReduce     int               //reduce分块数量
	mapfiles    chan string       // 文件队列
	reducefiles []chan string     // 文件队列
	tasks       map[int]*TaskItem // 实时任务表
	progress    string            // 进度 map/reduce
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

	switch c.progress {
	case "map":
		// 分配map任务
		if len(c.mapfiles) > 0 {
			reply.TaskType = MapTask
			ch = c.mapfiles
		} else {
			reply.TaskType = WaitTask
			return nil
		}
	case "reduce":
		{
			if idx := c.getReduceTask(); idx != -1 {
				// 分配reduce任务
				reply.TaskType = ReduceTask
				ch = c.reducefiles[idx]
			} else {
				reply.TaskType = WaitTask
				return nil
			}
		}
	default:
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
	mytask = &TaskItem{TaskID: reply.TaskID, Files: files, Completed: false, TaskType: reply.TaskType}
	c.tasks[mytask.TaskID] = mytask // 记录任务
	// 超时处理
	time.AfterFunc(10*time.Second, func() {
		if mytask.Completed {
			return
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		log.Printf("timeout: %d\n", mytask.TaskID)
		// 超时，重新分配任务
		if mytask.TaskType == MapTask {
			for _, file := range mytask.Files {
				c.mapfiles <- file
			}
		} else if mytask.TaskType == ReduceTask {
			c.makeReduce(mytask.Files, -1)
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

// makeReduce 根据文件名分配reduce任务，筛选mapid一致的文件
func (c *Coordinator) makeReduce(filenames []string, mapid int) {
	pattern := regexp.MustCompile(`^mr-(\d+)-(\d+)$`)
	for _, filename := range filenames {
		matches := pattern.FindStringSubmatch(filename)
		if len(matches) == 3 {
			mid, _ := strconv.Atoi(matches[1])
			if mapid != -1 && mid != mapid {
				continue
			}
			rid, _ := strconv.Atoi(matches[2])
			if rid < c.nReduce {
				c.reducefiles[rid] <- filename
			}
		}
	}
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

	if args.Err != nil {
		// 任务失败，重新添加文件
		if args.TaskType == MapTask {
			for _, file := range task.Files {
				c.mapfiles <- file
			}
		} else if args.TaskType == ReduceTask {
			c.makeReduce(task.Files, -1)
		}
		return nil
	}
	task.Completed = true
	delete(c.tasks, args.TaskID) // 删除任务，之后重新分配
	// 完成map任务，创建reduce任务需要的文件
	if args.TaskType == MapTask {
		dir, _ := os.Getwd()
		// 获取符合条件的文件名，用作reduce任务的输入
		filenames := make([]string, 0)
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				filenames = append(filenames, info.Name())
			}
			return nil
		})
		c.makeReduce(filenames, args.TaskID)
	}
	if len(c.tasks) == 0 && len(c.mapfiles) == 0 {
		// 所有map任务完成，开始reduce任务
		c.progress = "reduce"
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
	reduced := c.progress == "reduce"
	for _, ch := range c.reducefiles {
		if len(ch) > 0 {
			reduced = false
			break
		}
	}
	isdone := len(c.mapfiles) == 0 && reduced && len(c.tasks) == 0
	if isdone {
		log.Printf("!!!!!!!!!!!!!!!!!!!!! coordinator done!!!!!!!!!!\n")
	}
	// 没有待处理文件 且 没有任务在运行
	return isdone
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	log.SetPrefix("c_" + fmt.Sprint(os.Getpid()) + " ")
	log.SetFlags(0)
	c := Coordinator{}
	log.Printf("coordinator: reduce %d, files %d\n", nReduce, len(files))
	// Your code here.
	c.mu = sync.Mutex{}
	c.nReduce = nReduce
	c.mapfiles = make(chan string, len(files))
	c.progress = "map" // wait for workers to register
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
