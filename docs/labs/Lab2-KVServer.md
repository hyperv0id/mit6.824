## 介绍

在本实验中，您将构建一个单机键/值服务器，确保即使在网络故障的情况下，每个操作也能被精确执行一次，并且这些操作是[线性化](https://pdos.csail.mit.edu/6.824/papers/linearizability-faq.txt)的。后续实验将复制此类服务器以应对服务器崩溃的情况。

客户端可以向键/值服务器发送三种不同的 RPC： `Put(key, value)` , `Append(key, arg)` 和 `Get(key)` 。服务器维护着 键/值对的内存映射。键和值是字符串。 `Put(key, value)` 安装或替换映射中特定键的值， `Append(key, arg)` 将参数附加到键的值*并*返回旧值， `Get(key)` 获取键的当前值。对于不存在的键， `Get` 应返回空字符串。对于不存在的键， `Append` 应表现得好像现有值为零长度字符串。每个客户端通过具有 Put/Append/Get 方法的 `Clerk` 与服务器通信。 `Clerk` 管理与服务器的 RPC 交互。

您的服务器必须确保对 `Clerk` 的 Get/Put/Append 方法的应用调用是可线性化的。如果客户端请求不是并发的，每个客户端的 Get/Put/Append  调用应观察到由先前调用序列所隐含的状态修改。对于并发调用，返回值和最终状态必须与操作按某种顺序一次执行一个时相同。如果调用在时间上重叠，则它们是并发的：例如，如果客户端 X 调用 `Clerk.Put()` ，客户端 Y 调用 `Clerk.Append()` ，然后客户端 X 的调用返回。一个调用必须观察到在调用开始之前所有已完成调用的效果。

线性一致性对应用程序来说很方便，因为它类似于单个服务器一次处理一个请求的行为。例如，如果一个客户端从服务器获得了更新请求的成功响应，那么随后从其他客户端发起的读取操作保证能看到该更新的效果。对于单个服务器来说，提供线性一致性相对容易。

## 入门

我们为您提供 `src/kvsrv` 中的骨架代码和测试。您需要修改 `kvsrv/client.go` 、 `kvsrv/server.go` ，以及 `kvsrv/common.go` .

要开始运行，请执行以下命令。别忘了使用 `git pull` 来获取最新软件。

```
$ cd ~/6.5840
$ git pull
...
$ cd src/kvsrv
$ go test
...
$
```

## 任务

### 无网络故障的键/值服务器（[简单](https://pdos.csail.mit.edu/6.824/labs/guidance.html)）

> 你的第一个任务是实现一个在**没有消息丢失**时能正常工作的解决方案。
>
> 你需要在 `client.go` 中的 Clerk Put/Append/Get 方法中添加 RPC 发送代码，并实现 `Put` 、 `Append()` 和 `Get()` 中的 RPC 处理程序 `server.go` .
>
> 当您通过测试套件中的前两个测试：“one client”和“many clients”时，您就完成了此任务。
>
> 记得检查你的代码是否无竞争条件 `go test -race` .

### 带丢包消息的键/值服务器（[简单](https://pdos.csail.mit.edu/6.824/labs/guidance.html)）

现在你应该修改你的解决方案，以在面对丢弃时继续 消息（例如，RPC 请求和 RPC 回复）。 如果消息丢失，客户端的 `ck.server.Call()` 将返回 `false` （更准确地说， `Call()` 等待） 等待超时间隔内的回复消息，并返回 false 如果在该时间内未收到回复）。 你将面临的一个问题是 `Clerk` 可能不得不多次发送 RPC，直到它 成功。每次调用 `Clerk.Put()` 或 `Clerk.Append()` ，然而，应该只导致一次*单一*执行，因此您必须确保 重新发送不会导致服务器执行 请求两次。

> 添加代码到 `Clerk` 以在未收到回复时重试， 和 `server.go` 如果操作需要，用于过滤重复项。这些说明包括关于[重复检测](https://pdos.csail.mit.edu/6.824/notes/l-raft-QA.txt)的指导。

### 提示

- 您需要唯一标识客户端操作，以确保键/值服务器仅执行每个操作一次。
- 必须仔细考虑服务器必须维护什么状态来处理重复的 `Get()` 、 `Put()` 和 `Append()` 请求（如果有的话）。
- 您的重复检测方案应快速释放服务器内存，例如通过让每个 RPC 暗示客户端已看到其前一个 RPC 的回复。可以假设客户端一次只会向 Clerk 发起一个调用。

您的代码现在应该通过所有测试，如下所示：

```sh
$ go test
Test: one client ...
  ... Passed -- t  3.8 nrpc 31135 ops 31135
Test: many clients ...
  ... Passed -- t  4.7 nrpc 102853 ops 102853
Test: unreliable net, many clients ...
  ... Passed -- t  4.1 nrpc   580 ops  496
Test: concurrent append to same key, unreliable ...
  ... Passed -- t  0.6 nrpc    61 ops   52
Test: memory use get ...
  ... Passed -- t  0.4 nrpc     4 ops    0
Test: memory use put ...
  ... Passed -- t  0.2 nrpc     2 ops    0
Test: memory use append ...
  ... Passed -- t  0.4 nrpc     2 ops    0
Test: memory use many puts ...
  ... Passed -- t 11.5 nrpc 100000 ops    0
Test: memory use many gets ...
  ... Passed -- t 12.2 nrpc 100001 ops    0
PASS
ok      6.5840/kvsrv    39.000s
```

每个 `Passed` 后的数字分别是实时秒数、发送的 RPC 数量（包括客户端 RPC）以及执行的键/值操作数量（ `Clerk` Get/Put/Append 调用）。