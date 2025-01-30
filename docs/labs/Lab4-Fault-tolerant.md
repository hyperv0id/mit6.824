# 实验 3：Raft

## 介绍

这是系列实验中的第一个，你将构建一个容错的键/值存储系统。在本实验中，你将实现 Raft，一种复制状态机协议。在下一个实验中，你将在 Raft 之上构建一个键/值服务。然后，你将通过多个复制状态机“分片”你的服务，以实现更高的性能。

复制服务通过在多个副本服务器上存储其状态（即数据）的完整副本来实现容错。即使某些服务器发生故障（崩溃或网络中断或不稳定），复制也能使服务继续运行。挑战在于，故障可能导致副本持有不同的数据副本。

Raft  将客户端请求组织成一个序列，称为日志，并确保所有副本服务器看到相同的日志。每个副本按照日志顺序执行客户端请求，将其应用到服务状态的本地副本中。由于所有存活的副本都看到相同的日志内容，它们以相同的顺序执行相同的请求，从而继续保持相同的服务状态。如果服务器发生故障但随后恢复，Raft 会负责将其日志更新至最新状态。只要至少大多数服务器存活并能相互通信，Raft 就会继续运行。如果没有这样的多数，Raft  将无法取得进展，但一旦大多数服务器能够再次通信，它将从上次中断的地方继续。

在本实验中，您将实现 Raft 作为一个 Go 对象类型及其相关方法，旨在作为更大服务中的一个模块使用。一组 Raft 实例通过 RPC 相互通信，以维护复制的日志。您的 Raft 接口将支持一个无限序列的编号命令，也称为日志条目。这些条目通过*索引号*进行编号。具有给定索引的日志条目最终将被提交。此时，您的 Raft 应将日志条目发送给更大的服务以执行。

您应遵循设计中的 [扩展的 Raft 论文](https://pdos.csail.mit.edu/6.824/papers/raft-extended.pdf)，特别关注图 2。你将实现论文中的大部分内容，包括保存持久状态以及在节点失败并重新启动后读取它。你将不会实现集群成员变更（第 6 节）。

本实验分为四个部分。你必须在相应的截止日期前提交每个部分。

##  开始

如果你已经完成了实验 1，那么你已经拥有了一份实验源代码的副本。如果没有，你可以在[实验 1 说明](https://pdos.csail.mit.edu/6.824/labs/lab-mr.html)中找到通过 git 获取源代码的指导。

我们为您提供骨架代码 `src/raft/raft.go` 。我们还提供了一组测试，您应使用这些测试来推动您的实现工作，我们也将使用这些测试来评估您提交的实验。测试位于 `src/raft/test_test.go` 。

在评分您的提交时，我们将不带[ `-race` 标志](https://go.dev/blog/race-detector)运行测试。然而，在开发解决方案的过程中，您应通过使用 `-race` 标志运行测试来确保代码不存在竞态条件。

要开始运行，请执行以下命令。别忘了使用 `git pull` 来获取最新软件。

```sh
1
$ cd ~/6.5840
2
$ git pull
3
...
4
$ cd src/raft
5
$ go test
6
Test (3A): initial election ...
7
--- FAIL: TestInitialElection3A (5.04s)
8
        config.go:326: expected one leader, got none
9
Test (3A): election after network failure ...
10
--- FAIL: TestReElection3A (5.03s)
11
        config.go:326: expected one leader, got none
12
...
13
$
```

### 代码

通过添加代码实现 Raft `raft/raft.go` . 在该文件中，您会发现 骨架代码，以及如何发送和接收的示例 RPCs.

 您的实现必须支持以下接口，测试程序以及（最终）您的键/值服务器将使用该接口。您可以在 `raft.go` 的注释中找到更多详细信息。

```go
1
// create a new Raft server instance:
2
rf := Make(peers, me, persister, applyCh)
3
​
4
// start agreement on a new log entry:
5
rf.Start(command interface{}) (index, term, isleader)
6
​
7
// ask a Raft for its current term, and whether it thinks it is leader
8
rf.GetState() (term, isLeader)
9
​
10
// each time a new entry is committed to the log, each Raft peer
11
// should send an ApplyMsg to the service (or tester).
12
type ApplyMsg
```

服务调用 `Make(peers,me,…)` 来创建一个 Raft 对等体。peers 参数是一个网络标识符数组 Raft 对等节点（包括此节点）的数量，用于 RPC。 `me` 参数是此对等方在 peers 数组中的索引。 `Start(command)` 要求 Raft 开始处理，将命令附加到复制的日志中。 `Start()`  应立即返回，无需等待日志追加 完成。服务期望您的实现发送一个 `ApplyMsg` 对于每个新提交的日志条目 `applyCh` 通道参数传递给 `Make()` 。

`raft.go` 包含发送 RPC（ `sendRequestVote()` ）和处理传入 RPC（ `RequestVote()` ）的示例代码。你的 Raft 节点应使用 labrpc Go 包（源代码在 `src/labrpc` ）交换 RPC。测试程序可以指示 `labrpc` 延迟 RPC、重新排序或丢弃它们，以模拟各种网络故障。虽然你可以暂时修改 `labrpc` ，但请确保你的 Raft 能在原始的 `labrpc` 上运行，因为我们将使用它来测试和评分你的实验。你的 Raft 实例只能通过 RPC 进行交互；例如，不允许使用共享的 Go 变量或文件进行通信。

后续实验将基于本实验进行，因此给自己足够的时间编写稳健的代码非常重要。

## 任务

### 第 3A 部分：领导者选举（[中等](https://pdos.csail.mit.edu/6.824/labs/guidance.html)）

实现 Raft 领导者选举和心跳（ `AppendEntries` 无日志条目的 RPC）。第 3A 部分的目标是选出一个单一的领导者，在没有故障的情况下保持领导者地位，以及如果旧领导者失败或与旧领导者之间的数据包丢失，则由新领导者接管。运行 `go test -run 3A ` 来测试你的 3A 代码。

- 你不能直接轻松地运行你的 Raft 实现；相反，你应该通过测试器来运行它，即 `go test -run 3A ` 。
- 遵循论文中的图 2。此时，您需要关注发送和接收 RequestVote RPC、与选举相关的服务器规则，以及与领导者选举相关的状态。
- 将图 2 中的领导者选举状态添加到 `Raft` 结构体中的 `raft.go` 。你还需要定义一个结构体来保存每个日志条目的信息。
- 填写 `RequestVoteArgs` 和 `RequestVoteReply` 结构体。修改 `Make()` 创建一个后台 goroutine，当它一段时间没有收到其他对等方的消息时，通过发送 `RequestVote` RPC 定期启动领导者选举。实现 `RequestVote()` RPC 处理程序，以便服务器可以相互投票。
- 要实现心跳机制，定义一个 `AppendEntries` RPC 结构体（尽管你可能不会 需要所有参数），并让领导者发送 定期将它们写出来。写一个 `AppendEntries` RPC 处理方法。
- 测试人员要求领导者每秒发送心跳 RPC 不超过十次。
- 测试者要求你的 Raft 在旧领导者失败后的五秒内选举出新的领导者（如果大多数节点仍能通信）。
- 论文的第 5.2 节提到了选举超时时间在 150 到 300 毫秒之间。这样的范围只有在领导者发送心跳的频率远高于每 150 毫秒一次（例如，每 10  毫秒一次）时才有意义。由于测试程序限制你每秒只能发送几十次心跳，你将不得不使用比论文中 150 到 300  毫秒更大的选举超时时间，但也不能太大，因为那样你可能无法在五秒内选出领导者。
- 你可能会发现 Go 的 [rand](https://golang.org/pkg/math/rand/) 有用。
- 你需要编写代码来定期执行操作或 经过时间延迟后。最简单的方法是创建 一个带有循环调用的 goroutine [time.Sleep()](https://golang.org/pkg/time/#Sleep)；请参阅为此目的创建的 `ticker()` goroutine。不要使用 Go 的 `time.Timer` 或 `time.Ticker` ，它们难以正确使用。
- 如果你的代码难以通过测试，请再次阅读论文的图 2；领导者选举的完整逻辑分布在图的多个部分中。
- 不要忘记实现 `GetState()` 。
- 测试者在调用你的 Raft 的 `rf.Kill()` 时 永久关闭一个实例。您可以检查是否 `Kill()` 已使用 `rf.killed()` 调用。你可能希望在所有循环中执行此操作，以避免死掉的 Raft 实例打印出令人困惑的消息。
- Go RPC 仅发送名称以大写字母开头的结构体字段。子结构体也必须具有大写的字段名（例如，数组中日志记录的字段）。 `labgob` 包会对此发出警告；不要忽略这些警告。
- 本实验最具挑战性的部分可能是调试。花些时间让你的实现易于调试。请参考[指南](https://pdos.csail.mit.edu/6.824/labs/guidance.html)页面获取调试技巧。

在提交第 3A 部分之前，请确保您通过了 3A 测试，以便看到类似这样的内容：

```
1
$ go test -run 3A
2
Test (3A): initial election ...
3
  ... Passed --   3.5  3   58   16840    0
4
Test (3A): election after network failure ...
5
  ... Passed --   5.4  3  118   25269    0
6
Test (3A): multiple elections ...
7
  ... Passed --   7.3  7  624  138014    0
8
PASS
9
ok      6.5840/raft 16.265s
10
$
```

每个“Passed”行包含五个数字；这些分别是测试所花费的时间（秒）、Raft 对等体的数量、测试期间发送的 RPC 数量、RPC 消息中的总字节数以及 Raft  报告已提交的日志条目数。您的数字将与此处显示的不同。如果您愿意，可以忽略这些数字，但它们可能有助于您检查实现发送的 RPC  数量是否合理。对于所有实验 3、4 和 5，如果所有测试的总时间超过 600 秒（ `go test` ），或者任何单个测试超过 120 秒，评分脚本将使您的解决方案失败。

当我们评分您的提交时，我们将在没有[ `-race` 标志](https://go.dev/blog/race-detector)的情况下运行测试。然而，您应确保您的代码在使用 `-race` 标志时始终通过测试。

### 第 3B 部分：日志（[困难](https://pdos.csail.mit.edu/6.824/labs/guidance.html)）

实现领导者和跟随者代码以追加新的日志条目，以便 `go test -run 3B ` 测试通过。

- 运行 `git pull` 以获取最新的实验室软件。
- 你的首要目标应该是通过 `TestBasicAgree3B()` 。首先实现 `Start()` ，然后编写代码通过 `AppendEntries` RPC 发送和接收新的日志条目，遵循图 2。在每个对等体上通过 `applyCh` 发送每个新提交的条目。
- 您需要实现选举限制（论文中的第 5.4.1 节）。
- 您的代码可能包含循环，这些循环会反复检查某些事件。 不要有这些循环 持续执行而不暂停，因为 会减慢你的实现速度，导致测试失败。 使用 Go 的 [条件变量](https://golang.org/pkg/sync/#Cond), 或插入一个 `time.Sleep(10 * time.Millisecond)` 在每次循环迭代中。
- 为了未来的实验，请帮自己一个忙，编写（或重写）干净清晰的代码。如需灵感，请重新访问我们的[指南页面](https://pdos.csail.mit.edu/6.824/labs/guidance.html)，获取有关如何开发和调试代码的建议。
- 如果你测试失败，看看 `test_test.go` 和 `config.go` 用于理解正在测试的内容。 `config.go` 还展示了测试人员如何使用 Raft API。

即将进行的实验室测试可能会因为代码运行过慢而失败。你可以使用`time`命令检查你的解决方案使用了多少实际时间和 CPU 时间。以下是典型输出：

```
1
$ time go test -run 3B
2
Test (3B): basic agreement ...
3
  ... Passed --   0.9  3   16    4572    3
4
Test (3B): RPC byte count ...
5
  ... Passed --   1.7  3   48  114536   11
6
Test (3B): agreement after follower reconnects ...
7
  ... Passed --   3.6  3   78   22131    7
8
Test (3B): no agreement if too many followers disconnect ...
9
  ... Passed --   3.8  5  172   40935    3
10
Test (3B): concurrent Start()s ...
11
  ... Passed --   1.1  3   24    7379    6
12
Test (3B): rejoin of partitioned leader ...
13
  ... Passed --   5.1  3  152   37021    4
14
Test (3B): leader backs up quickly over incorrect follower logs ...
15
  ... Passed --  17.2  5 2080 1587388  102
16
Test (3B): RPC counts aren't too high ...
17
  ... Passed --   2.2  3   60   20119   12
18
PASS
19
ok      6.5840/raft 35.557s
20
​
21
real    0m35.899s
22
user    0m2.556s
23
sys 0m1.458s
24
$
```

"ok 6.5840/raft 35.557s" 表示 Go 测量了 3B 所花费的时间 测试耗时 35.557 秒实际（挂钟）时间。"用户 0m2.556s" 表示代码消耗了 2.556 秒的 CPU 时间，或 实际执行指令所花费的时间（而不是等待或） 睡觉）。如果你的解决方案使用了远超过一分钟的实际时间 对于 3B 测试，或超过 5 秒的 CPU 时间，您可能会运行 以后会遇到麻烦。寻找花费在睡眠或等待 RPC 上的时间。 超时、循环运行而不休眠或等待条件或 频道消息，或发送的大量 RPC。

### 第 3C 部分：持久性（[困难](https://pdos.csail.mit.edu/6.824/labs/guidance.html)）

如果基于 Raft 的服务器重启，它应该从离开的地方恢复服务。这要求 Raft 保持持久状态，以便在重启后仍能存活。论文的图 2 提到了哪些状态应该是持久的。

一个实际的实现会在每次 Raft 的持久状态发生变化时将其写入磁盘，并在重启后从磁盘读取状态。你的实现不会使用磁盘；相反，它将从一个 `Persister` 对象保存和恢复持久状态（参见 `persister.go` ）。调用 `Raft.Make()` 的人会提供一个 `Persister` 。 最初保存 Raft 最近持久化状态（如果 any). Raft 应该从该状态初始化其状态 `Persister` ，并应使用它来保存其持久状态，每次状态更改时都应如此。使用 `Persister` 的 `ReadRaftState()` 和 `Save()` 方法。

完成函数 `persist()`  和 `readPersist()` 在 `raft.go` 中通过添加代码来保存和恢复持久状态。您需要将状态编码（或“序列化”）为字节数组，以便将其传递给 `Persister` 。使用 `labgob` 编码器；请参阅 `persist()` 和 `readPersist()` 中的注释。 `labgob` 类似于 Go 的 `gob` 编码器，但如果你尝试编码带有小写字段名的结构体，它会打印错误信息。目前，将 `nil` 作为第二个参数传递给 `persister.Save()` 。在你的实现更改持久状态的地方插入对 `persist()` 的调用。一旦你完成了这些操作，并且如果你的实现其余部分是正确的，你应该能通过所有的 3C 测试。

您可能需要一种优化，即一次备份多个条目的 nextIndex。请查看从第 7 页底部到第 8 页顶部（以灰色线标记）的[扩展 Raft 论文](https://pdos.csail.mit.edu/6.824/papers/raft-extended.pdf)。论文对细节描述较为模糊；您需要填补这些空白。一种可能性是让拒绝消息包含：

```
1
    XTerm:  term in the conflicting entry (if any)
2
    XIndex: index of first entry with that term (if any)
3
    XLen:   log length
```

那么领导者的逻辑可以是这样的：

```
1
  Case 1: leader doesn't have XTerm:
2
    nextIndex = XIndex
3
  Case 2: leader has XTerm:
4
    nextIndex = leader's last entry for XTerm
5
  Case 3: follower's log is too short:
6
    nextIndex = XLen
```

 其他一些提示：

- 运行 `git pull` 以获取最新的实验室软件。
- 3C 测试比 3A 或 3B 的要求更高，失败可能是由 3A 或 3B 代码中的问题引起的。

您的代码应通过所有 3C 测试（如下所示），以及 3A 和 3B 测试。

```
1
$ go test -run 3C
2
Test (3C): basic persistence ...
3
  ... Passed --   5.0  3   86   22849    6
4
Test (3C): more persistence ...
5
  ... Passed --  17.6  5  952  218854   16
6
Test (3C): partitioned leader and one follower crash, leader restarts ...
7
  ... Passed --   2.0  3   34    8937    4
8
Test (3C): Figure 8 ...
9
  ... Passed --  31.2  5  580  130675   32
10
Test (3C): unreliable agreement ...
11
  ... Passed --   1.7  5 1044  366392  246
12
Test (3C): Figure 8 (unreliable) ...
13
  ... Passed --  33.6  5 10700 33695245  308
14
Test (3C): churn ...
15
  ... Passed --  16.1  5 8864 44771259 1544
16
Test (3C): unreliable churn ...
17
  ... Passed --  16.5  5 4220 6414632  906
18
PASS
19
ok      6.5840/raft 123.564s
20
$
21
​
```

在提交之前多次运行测试并检查每次运行是否打印 `PASS` 是个好主意。

```
1
$ for i in {0..10}; do go test; done
```

### 第 3D 部分：日志压缩（[困难](https://pdos.csail.mit.edu/6.824/labs/guidance.html)）

就目前情况而言，重启的服务器会重放 完整的 Raft 日志以恢复其状态。然而，这不是 对于长时间运行的服务来说，记住完整的 Raft 日志是不切实际的 永远。相反，你将修改 Raft 以与那些服务协作 持久化存储其状态的“快照”，定期进行 Raft 丢弃快照之前的日志条目。 结果是更少的持久数据和更快的重启。 然而，现在有可能出现一个追随者落后太多的情况，以至于 领导者已丢弃了它需要追赶的日志条目； 领导者随后必须发送一个快照以及从该时间点开始的日志 快照。第 7 节的 [扩展的 Raft 论文](https://pdos.csail.mit.edu/6.824/papers/raft-extended.pdf)概述了该方案；您需要设计具体细节。

您的 Raft 必须提供以下功能，服务可以通过其状态的序列化快照调用：

```
1
Snapshot(index int, snapshot []byte)
```

在 Lab 3D 中，测试者会定期调用 `Snapshot()` 。在 Lab 4 中，你将编写一个键/值服务器，该服务器会调用 `Snapshot()` ；快照将包含完整的键/值对表。服务层会在每个对等节点（不仅仅是领导者）上调用 `Snapshot()` 。

 `index` 参数表示快照中反映的最高日志条目。Raft 应丢弃该点之前的日志条目。您需要修改 Raft 代码，使其在仅存储日志尾部的情况下运行。

你需要实现论文中讨论的 `InstallSnapshot` RPC，它允许 Raft 领导者告诉落后的 Raft 对等方用快照替换其状态。你可能需要仔细考虑 InstallSnapshot 应如何与图 2 中的状态和规则交互。

当跟随者的 Raft 代码接收到 InstallSnapshot RPC 时，它可以使用 `applyCh` 将快照发送到服务中的 `ApplyMsg` 。 `ApplyMsg` 结构体定义已经包含了您将需要的字段（也是测试者期望的）。请注意，这些快照只会推进服务的状态，而不会导致其回退。

如果服务器崩溃，它必须从持久化数据重新启动。您的 Raft 应同时持久化 Raft 状态及相应的快照。 使用第二个参数来 `persister.Save()` 保存快照。如果没有快照，将 `nil` 作为第二个参数传递。

当服务器重启时，应用层读取持久化的快照并恢复其保存的状态。

实现 `Snapshot()` 和 InstallSnapshot RPC，以及对 Raft 的更改以支持这些功能（例如，使用修剪日志的操作）。当您的解决方案通过 3D 测试（以及所有之前的 Lab 3 测试）时，即表示完成。

-   `git pull` 确保您拥有最新的软件。
- 一个好的起点是修改你的代码，使其能够仅存储从某个索引 X 开始的日志部分。最初，你可以将 X 设为零并运行 3B/3C 测试。然后让 `Snapshot(index)` 丢弃 `index` 之前的日志，并将 X 设置为 `index` 。如果一切顺利，你现在应该能通过第一个 3D 测试。
- 下一步：如果领导者没有所需的日志条目以使追随者保持最新状态，则让领导者发送一个 InstallSnapshot RPC。
- 将整个快照通过单个 InstallSnapshot RPC 发送。不要实现图 13 中用于分割快照的 `offset` 机制。
- Raft 必须以允许 Go 垃圾收集器释放和重用内存的方式丢弃旧的日志条目；这要求没有对已丢弃日志条目的可达引用（指针）。
- 完成全套 Lab 3 测试（3A+3B+3C+3D）而不使用 `-race` 时，合理的消耗时间为 6 分钟实际时间和 1 分钟 CPU 时间。使用 `-race` 运行时，大约需要 10 分钟实际时间和 2 分钟 CPU 时间。

您的代码应通过所有 3D 测试（如下所示），以及 3A、3B 和 3C 测试。

```
1
$ go test -run 3D
2
Test (3D): snapshots basic ...
3
  ... Passed --  11.6  3  176   61716  192
4
Test (3D): install snapshots (disconnect) ...
5
  ... Passed --  64.2  3  878  320610  336
6
Test (3D): install snapshots (disconnect+unreliable) ...
7
  ... Passed --  81.1  3 1059  375850  341
8
Test (3D): install snapshots (crash) ...
9
  ... Passed --  53.5  3  601  256638  339
10
Test (3D): install snapshots (unreliable+crash) ...
11
  ... Passed --  63.5  3  687  288294  336
12
Test (3D): crash and restart all servers ...
13
  ... Passed --  19.5  3  268   81352   58
14
PASS
15
ok      6.5840/raft      293.456s
```

