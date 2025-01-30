## 介绍

在本实验中，您将构建一个 MapReduce 系统。您将实现一个工作进程，该进程调用应用程序的 Map 和 Reduce  函数，并处理文件的读写；以及一个协调器进程，该进程向工作进程分配任务并处理失败的工作进程。您将构建类似于 [MapReduce  论文](http://research.google.com/archive/mapreduce-osdi04.pdf)中的内容。（注意：本实验使用“coordinator”而非论文中的“master”。）

## 开始

你需要[设置好 Go 环境](https://pdos.csail.mit.edu/6.824/labs/go.html) 来完成实验。

获取初始实验室软件 [git](https://git-scm.com/)（一个版本控制系统）。 要了解更多关于 git 的信息，请查看 [Pro Git 书籍](https://git-scm.com/book/en/v2) 或 [git 用户手册](http://www.kernel.org/pub/software/scm/git/docs/user-manual.html)。

我们在`src/main/mrsequential.go`中为您提供了一个简单的**顺序 mapreduce** 实现  . 它在一个进程中逐个运行映射和归约。我们还为您提供了几个 MapReduce 应用程序： `mrapps/wc.go` 中的单词计数和 `mrapps/indexer.go` 中的文本索引器。您可以按如下方式顺序运行单词计数：

```
$ cd ~/6.5840
$ cd src/main
$ go build -buildmode=plugin ../mrapps/wc.go
$ rm mr-out*
$ go run mrsequential.go wc.so pg*.txt
$ more mr-out-0
A 509
ABOUT 2
ACT 8
...
```

`mrsequential.go` 将其输出留在文件 `mr-out-0` 中。输入来自名为 `pg-xxx.txt` 的文本文件。

随意复用 `mrsequential.go` 的代码。你也应该看看 `mrapps/wc.go` ，了解 MapReduce 应用程序代码的样子。

对于本次实验及所有其他实验，我们可能会对我们提供的代码进行更新。为了确保您能够获取这些更新并使用 `git pull` 轻松合并它们，最好将我们提供的代码保留在原始文件中。您可以根据实验说明中的指示添加我们提供的代码；只是不要移动它。将您自己的新函数放在新文件中是可以的。

## 你的工作（[中等/困难](https://pdos.csail.mit.edu/6.824/labs/guidance.html)）

你的任务是实现一个分布式 MapReduce，包括 两个程序，协调者和工作者。将会 仅一个协调器进程，以及一个或多个工作进程在执行中 并行。在真实系统中，工作者会在一组 不同的机器，但在本实验中，您将在单台机器上运行它们。 工人们将通过 RPC 与协调员交谈。每个工作进程将， 在循环中询问 任务协调员从一个或多个文件中读取任务的输入， 执行任务，将任务的输出写入一个 或多个文件，并再次向协调员请求一个 新任务。协调者应注意是否有工作者未完成任务。 它在合理的时间内完成任务（对于本实验，时限为**十秒**），并将相同的任务分配给不同的工作人员。

我们已经为您提供了一些代码来帮助您入门。协调者和工作者的“main”例程位于 `main/mrcoordinator.go` 和 `main/mrworker.go` 中；请不要更改这些文件。您应该将您的实现放在 `mr/coordinator.go` 中。 `mr/worker.go` ，和 `mr/rpc.go` 。

以下是如何在 word-count MapReduce 应用程序上运行你的代码。首先，**确保 word-count 插件已全新构建**：

```
$ go build -buildmode=plugin ../mrapps/wc.go
```

在 `main` 目录中，运行`mrcoordinator`。

```
$ rm mr-out*
$ go run mrcoordinator.go pg-*.txt
```

 `pg-*.txt` 的参数是 `mrcoordinator.go`  输入文件；每个文件对应一个“分割”，并且是 输入到一个 Map 任务。

在一个或多个其他窗口中，运行一些工作进程：

```
$ go run mrworker.go wc.so
```

当工人和协调员完成时，查看输出 在 `mr-out-*` 。当你完成实验后， 输出文件的排序并集应与顺序匹配 输出，像这样：

```
$ cat mr-out-* | sort | more
A 509
ABOUT 2
ACT 8
...
```

我们为您提供了一个测试脚本 `main/test-mr.sh` 。测试会检查当输入 `pg-xxx.txt` 文件时， `wc` 和 `indexer` MapReduce 应用程序是否产生正确的输出。测试还会检查您的实现是否并行运行 Map 和 Reduce 任务，以及您的实现是否能在运行任务时从崩溃的工作节点中恢复。

如果你现在运行测试脚本，它会挂起，因为协调器永远不会完成：

```
$ cd ~/6.5840/src/main
$ bash test-mr.sh
*** Starting wc test.
```

您可以在 `mr/coordinator.go` 的 Done 函数中将 `ret := false` 更改为 true，以便协调器立即退出。然后：

```
$ bash test-mr.sh
*** Starting wc test.
sort: No such file or directory
cmp: EOF on mr-wc-all
--- wc output is not the same as mr-correct-wc.txt
--- wc test: FAIL
$
```

测试脚本期望看到输出文件名为 `mr-out-X` ，每个 reduce 任务一个文件。 `mr/coordinator.go` 和 `mr/worker.go` 的空实现不会生成这些文件（或做其他任何事情），因此测试失败。


完成后，测试脚本的输出应如下所示：

```
$ bash test-mr.sh
*** Starting wc test.
--- wc test: PASS
*** Starting indexer test.
--- indexer test: PASS
*** Starting map parallelism test.
--- map parallelism test: PASS
*** Starting reduce parallelism test.
--- reduce parallelism test: PASS
*** Starting job count test.
--- job count test: PASS
*** Starting early exit test.
--- early exit test: PASS
*** Starting crash test.
--- crash test: PASS
*** PASSED ALL TESTS
$
```

You may see some errors from the Go RPC package that look like 

```
2019/12/16 13:27:09 rpc.Register: method "Done" has 1 input parameters; needs exactly three
```

忽略这些消息；将协调器注册为[RPC 服务器](https://golang.org/src/net/rpc/server.go)会检查其所有 方法适用于 RPC（有 3 个输入）；我们知道 `Done` 不是通过 RPC 调用的。

此外，根据您终止工作进程的策略，您可能会看到一些形式为

```
2024/02/11 16:21:32 dialing:dial unix /var/tmp/5840-mr-501: connect: connection refused
```

每个测试中出现少量这些消息是正常的；当工作进程无法联系协调器 RPC 服务器时，它们就会出现 协调员已退出。

### 几条规则：

- map 阶段应将中间键划分为多个桶以供 `nReduce` 个 reduce 任务，其中 `nReduce` 是 reduce 任务的数量 -- 该参数 `main/mrcoordinator.go` 传递给 `MakeCoordinator()` 。每个映射器应创建 `nReduce` 个中间文件以供 reduce 任务使用。
- worker 实现应将第 X 个 reduce 任务的输出放入文件 `mr-out-X` 中。
- A `mr-out-X` 文件应包含每个 Reduce 函数输出的一行。该行应使用 Go `"%v %v"` 生成。 格式，使用键和值调用。请查看 `main/mrsequential.go`  对于注释为“this is the correct format”的行。 如果您的实现与此格式偏差过大，测试脚本将失败。
- 您可以修改 `mr/worker.go` 、 `mr/coordinator.go` 和 `mr/rpc.go` 。您可以暂时修改其他文件进行测试，但请确保您的代码能与原始版本一起工作；我们将使用原始版本进行测试。
- 将中间 Map 输出放在当前目录的文件中，以便稍后作为 Reduce 任务的输入读取。
-  `main/mrcoordinator.go` 期望 `mr/coordinator.go` 实现一个 `Done()` 当 MapReduce 作业完全完成时返回 true 的方法；此时， `mrcoordinator.go` 将退出。
-  当作业完全完成时，工作进程应退出。实现这一点的一个简单方法是使用 `call()` 的返回值：如果工作进程无法联系到协调器，它可以假设协调器因为作业完成而退出，因此工作进程也可以终止。根据你的设计，你可能会发现让协调器向工作进程发送一个“请退出”的伪任务也很有帮助。

### 提示

#### 开发与调试

[指南页面](https://pdos.csail.mit.edu/6.824/labs/guidance.html)提供了一些关于开发和调试的技巧。

#### 修改Worker

一种入门方法是修改 `mr/worker.go` 's `Worker()` 向协调者发送 RPC 请求任务。然后修改协调者以响应一个尚未开始的 map 任务的文件名。接着修改工作节点以读取该文件并调用应用程序的 Map 函数，如 `mrsequential.go` 所示。

#### 插件加载

应用程序的 Map 和 Reduce 函数在运行时使用 Go 插件包加载，这些函数来自文件名以 `.so` 结尾的文件。

#### 构建插件

如果你更改了 `mr/` 目录中的任何内容，你可能需要重新构建你使用的任何 MapReduce 插件，使用类似 `go build -buildmode=plugin ../mrapps/wc.go` 的命令

#### 共享文件系统

该实验室依赖于工人们共享一个文件系统。当所有工人在同一台机器上运行时，这很简单，但如果工人们在不同的机器上运行，则需要像 GFS 这样的全局文件系统。

#### 文件的命名

中间文件的合理命名约定为 `mr-X-Y` ，其中 X 是 Map 任务编号，Y 是 reduce 任务编号。

#### Map任务

工人的`Map`任务代码将需要一种存储中间数据的方式 文件中的键/值对，以便能够正确读取 在 `reduce` 任务期间。一种可能性是使用 Go 的 `encoding/json` 包。为了 将键/值对以 JSON 格式写入已打开的文件：

```
enc := json.NewEncoder(file)
for _, kv := ... {
    err := enc.Encode(&kv)
}
//..............
enc := json.NewEncoder(file)
for _, kv := ... {
err := enc.Encode(&kv)
```

并读取这样的文件：

```
dec := json.NewDecoder(file)
for {
	var kv KeyValue
	if err := dec.Decode(&kv); err != nil {
		break
	}
	kva = append(kva, kv)
}
dec := json.NewDecoder(file)
for {
	var kv KeyValue
	if err := dec.Decode(&kv); err != nil {
		break
	}
	kva = append(kva, kv)
}
```

#### map工作

你的工作器的映射部分可以使用 `ihash(key)` 函数（在 `worker.go` 中）为给定键选择 reduce 任务。

#### 参考`mrsequential.go`

你可以从 `mrsequential.go` 中剽窃一些代码，用于读取 Map 输入文件、在 Map 和 Reduce 之间排序中间键/值对，以及将 Reduce 输出存储在文件中。

#### 使用锁

协调器作为 RPC 服务器将是并发的；别忘了锁定共享数据。

#### 检查竞态

使用 Go 的竞态检测器， `go run -race` 。 `test-mr.sh` 开头有一条注释，告诉你如何使用 `-race` 运行它。当我们评分你的实验时，我们将 **不会** 使用竞态检测器。尽管如此，如果你的代码存在竞态，即使不使用竞态检测器，我们在测试时它也很可能会失败。

#### 等待

工作者有时需要等待，例如，在最后一个映射完成之前，减少操作无法开始。一种可能性是工作者定期向协调者请求工作，在每次请求之间使用 `time.Sleep()` 进行休眠。另一种可能性是协调者中的相关 RPC 处理程序拥有一个等待循环，使用 `time.Sleep()` 或 `sync.Cond` 。Go 在自己的线程中为每个 RPC 运行处理程序，因此一个处理程序正在等待的事实不会阻止协调者处理其他 RPC。

#### 协调工作

协调者无法可靠地区分崩溃的工作者、存活但因某种原因停滞的工作者，以及正在执行但速度过慢以致无用的工作者。你所能做的最好的办法是让协调者等待一段时间，然后放弃并将任务重新分配给另一个工作者。在本实验中，让协调者等待十秒钟；之后，协调者应假定工作者已经死亡（当然，它可能并未死亡）。

#### 备份任务

如果您选择实施备份任务（第 3.6 节），请注意，我们测试您的代码在工作者执行任务而不崩溃时不会安排多余的任务。备份任务应仅在相对较长的时间段（例如 10 秒）后安排。

- [ ] 备份任务应仅在相对较长的时间段（例如 10 秒）后安排。

#### 崩溃恢复

要测试崩溃恢复，您可以使用 `mrapps/crash.go`  应用程序插件。它在 Map 和 Reduce 函数中随机退出。



为确保在存在部分写入文件的情况下无人观察到 崩溃时，MapReduce 论文提到了使用临时文件的技巧 并在完全写入后原子性地重命名它。你可以使用 `ioutil.TempFile` （或 `os.CreateTemp` ，如果您运行的是 Go 1.17 或更高版本）以创建临时文件， `os.Rename`  原子性地重命名它。

#### 测试

 `test-mr.sh` 在子目录 `mr-tmp` 中运行其所有进程，因此如果出现问题并且您想查看中间或输出文件，请查看该目录。在测试失败后，可以随意将 `test-mr.sh` 临时修改为 `exit` ，这样脚本就不会继续测试（并覆盖输出文件）。



 `test-mr-many.sh` 连续运行 `test-mr.sh` 多次，您可能希望这样做以发现低概率的错误。它接受一个参数，即运行测试的次数。您不应并行运行多个 `test-mr.sh` 实例，因为协调器将重用相同的套接字，导致冲突。

#### gRPC

Go RPC 仅发送名称以大写字母开头的结构体字段。子结构体也必须具有大写的字段名。

调用 RPC `call()` 函数时， reply struct 应包含所有默认值。RPC 调用 应该看起来像这样：

```
reply := SomeType{}
call(..., &reply)
reply := SomeType{}
call(..., &reply)
```

在调用之前未设置回复的任何字段。如果你 传递具有非默认字段的回复结构，RPC 系统可能会静默返回错误值。

### 挑战练习

#### 自己的 MapReduce 应用程序

实现你自己的 MapReduce 应用程序（参见 `mrapps/*` 中的示例），例如，分布式 Grep（MapReduce 论文的第 2.3 节）。

#### 多机器部署

将您的 MapReduce 协调器和工作程序分别运行在不同的机器上，就像实际应用中那样。您需要设置 RPC 以通过 TCP/IP 进行通信，而不是 Unix 套接字（参见 `Coordinator.server()` 中被注释掉的行），并使用共享文件系统进行文件的读写。例如，您可以将 `ssh` 部署到多台机器上。 或者你可以租用几个 AWS 实例并使用 [S3](https://aws.amazon.com/s3/) 用于存储。

