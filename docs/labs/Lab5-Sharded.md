## [6.5840](https://pdos.csail.mit.edu/6.824/index.html) - 2024 年春季

# 6.5840 实验 5：分片键/值服务

  **[协作政策](https://pdos.csail.mit.edu/6.824/labs/collab.html)** // **[提交实验室](https://pdos.csail.mit.edu/6.824/labs/submit.html)** // **[设置 Go](https://pdos.csail.mit.edu/6.824/labs/go.html)** // **[指导](https://pdos.csail.mit.edu/6.824/labs/guidance.html)** // **[广场](https://piazza.com/mit/spring2024/65840)**

------

### 介绍

你可以基于自己的想法完成一个[最终项目](https://pdos.csail.mit.edu/6.824/project.html)，或者完成这个实验。

在本实验中，您将构建一个键/值存储系统，该系统将键“分片”或分区到一组副本组中。一个分片是键/值对的一个子集；例如，所有以“a”开头的键可能是一个分片，所有以“b”开头的键是另一个分片，等等。分片的原因是性能。每个副本组仅处理几个分片的放置和获取操作，并且这些组并行操作；因此，系统总吞吐量（每单位时间的放置和获取操作）随着组数的增加而按比例增加。

您的分片键/值存储将包含两个主要组件。首先，一组副本组。每个副本组负责一部分分片，使用 Raft  复制。第二个组件是“分片控制器”。分片控制器决定哪个副本组应服务于每个分片；此信息称为配置。配置会随时间变化。客户端查询分片控制器以找到键的副本组，而副本组查询控制器以了解要服务哪些分片。整个系统有一个分片控制器，作为使用 Raft 实现的容错服务。

分片存储系统必须能够在副本组之间转移分片。一个原因是某些组可能比其他组负载更重，因此需要移动分片以平衡负载。另一个原因是副本组可能会加入或离开系统：可能会添加新的副本组以增加容量，或者现有的副本组可能会下线进行维修或退役。

本实验中的主要挑战在于处理重新配置——即分片到组的分配变化。在一个副本组内，所有组成员必须就重新配置相对于客户端 Put/Append/Get 请求的发生时间达成一致。例如，一个 Put 请求可能与导致副本组不再负责持有该 Put  键的分片的重新配置几乎同时到达。组中的所有副本必须就 Put 是在重新配置之前还是之后发生达成一致。如果是在之前，Put  应生效，且分片的新所有者将看到其效果；如果是在之后，Put 不会生效，客户端必须在新所有者处重试。推荐的方法是让每个副本组使用 Raft  不仅记录 Put、Append 和 Get 的序列，还要记录重新配置的序列。你需要确保在任何时候，每个分片最多只有一个副本组在服务请求。

重新配置还需要副本组之间的交互。例如，在配置 10 中，组 G1 可能负责分片 S1。在配置 11 中，组 G2 可能负责分片 S1。在从 10 到 11 的重新配置过程中，G1 和 G2 必须使用 RPC 将分片 S1 的内容（键/值对）从 G1 移动到 G2。

仅允许使用 RPC 进行客户端与服务器之间的交互。例如，不允许您的服务器不同实例共享 Go 变量或文件。

本实验室使用“配置”一词来指代将分片分配给副本组的过程。这与 Raft 集群成员变更不同。您无需实现 Raft 集群成员变更。

本实验室的总体架构（一个配置服务和一组副本组）遵循与 Flat Datacenter Storage、BigTable、Spanner、FAWN、Apache  HBase、Rosebud、Spinnaker  等系统相同的通用模式。然而，这些系统在细节上与本实验室有许多不同，并且通常更为复杂和强大。例如，实验室不会动态调整每个 Raft  组中的对等节点集合；其数据和查询模型非常简单；分片的交接过程较慢，且不支持并发客户端访问。

您的 Lab 5 分片服务器、Lab 5 分片控制器和 Lab 4 kvraft 都必须使用相同的 Raft 实现。

### 入门

执行 `git pull` 以获取最新的实验室软件。

我们为您提供 `src/shardctrler` 和 `src/shardkv` 中的骨架代码和测试。

要开始运行，请执行以下命令：

```
$ cd ~/6.5840
$ git pull
...
$ cd src/shardctrler
$ go test
--- FAIL: TestBasic (0.00s)
        test_test.go:11: wanted 1 groups, got 0
FAIL
exit status 1
FAIL    shardctrler     0.008s
$
```

完成后，您的实现应通过所有测试 在 `src/shardctrler` 目录中，以及所有在 	 `src/shardkv` .

### 部分 A：控制器与静态分片（[简单](https://pdos.csail.mit.edu/6.824/labs/guidance.html)）

首先，你将在 `shardctrler/server.go` 和 `client.go` 中实现分片控制器，以及一个能够处理不变（静态）配置的分片键/值服务器。完成后，你的代码应通过 `shardctrler/` 目录中的所有测试，并且 `5A` 在 `shardkv/` 中的测试。

```
$ cd ~/6.5840/src/shardctrler
$ go test
Test: Basic leave/join ...
  ... Passed
Test: Historical queries ...
  ... Passed
Test: Move ...
  ... Passed
Test: Concurrent leave/join ...
  ... Passed
Test: Minimal transfers after joins ...
  ... Passed
Test: Minimal transfers after leaves ...
  ... Passed
Test: Multi-group join/leave ...
  ... Passed
Test: Concurrent multi leave/join ...
  ... Passed
Test: Minimal transfers after multijoins ...
  ... Passed
Test: Minimal transfers after multileaves ...
  ... Passed
Test: Check Same config on servers ...
  ... Passed
PASS
ok  	6.5840/shardctrler	5.863s
$
$ cd ../shardkv
$ go test -run 5A
Test (5A): static shards ...
  ... Passed
Test (5A): rejection ...
  ... Passed
PASS
ok      6.5840/shardkv  9.262s
$
```

shardctrler 管理一系列编号的配置。每个配置描述了一组副本组以及分片到副本组的分配。每当这种分配需要改变时，分片控制器会创建一个包含新分配的新配置。键/值客户端和服务器在想要了解当前（或过去）配置时，会联系 shardctrler。

您的实现必须支持 `shardctrler/common.go` 中描述的 RPC 接口，该接口由 `Join` 、 `Leave` 、 `Move` 和 `Query` RPC 组成。这些 RPC 旨在允许管理员（及测试）控制 shardctrler：添加新的副本组、删除副本组以及在副本组之间移动分片。

 `Join` RPC 由管理员用于添加新的副本组。其参数是一组从唯一且非零的副本组标识符（GID）到服务器名称列表的映射。shardctrler  应通过创建一个包含新副本组的新配置来响应。新配置应尽可能均匀地在所有组之间分配分片，并且应尽可能少地移动分片以实现这一目标。如果 GID  不是当前配置的一部分（即允许 GID 加入、离开，然后再次加入），shardctrler 应允许重复使用 GID。

 `Leave` RPC 的参数是之前加入的组的 GID 列表。shardctrler 应创建一个不包括这些组的新配置，并将这些组的分片分配给剩余的组。新配置应尽可能均匀地将分片分配给各组，并应尽可能少地移动分片以实现这一目标。

 `Move` 的 RPC 参数是一个分片编号和一个 GID。shardctrler 应该创建一个新的配置，将分片分配给该组。 `Move` 的目的是允许我们测试您的软件。在 `Move` 之后的 `Join` 或 `Leave` 可能会撤销 `Move` ，因为 `Join` 和 `Leave` 会重新平衡。

 `Query` RPC 的参数是一个配置编号。shardctrler 会回复具有该编号的配置。如果编号为-1 或大于已知的最大配置编号，shardctrler 应回复最新的配置。 `Query(-1)` 的结果应反映 shardctrler 在接收到 `Query(-1)` RPC 之前完成处理的每个 `Join` 、 `Leave` 或 `Move` RPC。

第一个配置应编号为零。它不应包含任何组，所有分片都应分配给 GID 零（一个无效的 GID）。下一个配置（响应 `Join` RPC 创建）应编号为 1，依此类推。通常，分片数量会显著多于组数量（即每个组将服务多个分片），以便负载可以在相当细的粒度上进行调整。

你必须在 `shardctrler/` 目录中实现 `client.go` 和 `server.go` 中指定的接口。你的 shardctrler 必须具有容错性，使用你在 Lab 3/4 中开发的 Raft 库。当你通过 `shardctrler/` 中的所有测试时，即完成了此任务。

- 从你的 kvraft 服务器的一个精简副本开始。
- 您应实施重复客户端请求检测 用于分片控制器的 RPC。shardctrler 测试 不要测试这个，但 shardkv 测试稍后会用到 你的 shardctrler 在一个不可靠的网络上；你可能 如果您的 shardctrler 无法通过 shardkv 测试 不过滤重复的 RPCs。
- 执行分片重新平衡的状态机代码需要是确定性的。在 Go 中，map 的迭代顺序是[不确定的](https://blog.golang.org/maps#TOC_7.)。
- Go 中的映射是引用类型。如果你将一个映射类型的变量赋值给另一个变量，这两个变量将引用同一个映射。因此，如果你想基于一个已有的映射创建一个新的 `Config` ，你需要创建一个新的映射对象（使用 `make()` ），并逐个复制键和值。
- Go 竞争检测器（go test -race）可能帮助你发现 bug。

接下来，在 `shardkv/` 目录中，实现一个足够的分片键/值服务器，以通过 `shardkv/` 中的前两个测试。再次从您现有的 `kvraft` 服务器复制代码开始。您应该能够在不做任何关于分片的特殊处理的情况下通过第一个测试，因为我们提供的 `shardkv/client.go` 会负责将 RPC 发送到控制器分配给相关键的组。

对于第二次 `shardkv` 测试，每个 k/v 副本组必须拒绝针对其未分配分片的键的请求。此时，k/v 服务器定期向控制器询问最新配置，并在每次客户端 Get/Put/Append RPC 到达时检查该配置即可。使用 `key2shard()` （在 `client.go` 中）来查找键的分片编号。

您的服务器应对客户端 RPC 响应一个 `ErrWrongGroup` 错误，当该 RPC 的键值不属于服务器负责范围时（即，键值所在的分片未分配给服务器所在组）。

您的服务器不应调用分片控制器的 `Join()` 处理程序。测试器将在适当时调用 `Join()` 。

​		

### 部分 B：分片移动 ([困难](https://pdos.csail.mit.edu/6.824/labs/guidance.html))

执行 `git pull` 以获取最新的实验室软件。

实验室这一部分的主要任务是在控制器更改分片时，在副本组之间移动分片，并以提供线性化 k/v 客户端操作的方式进行。

每个分片只需要在其 Raft 副本组中的大多数服务器存活且能够相互通信，并且能够与 `shardctrler` 服务器中的大多数通信时取得进展。即使某些副本组中的少数服务器死亡、暂时不可用或运行缓慢，您的实现也必须能够运行（处理请求并能够根据需要重新配置）。

一个 shardkv 服务器仅属于一个副本组。给定副本组中的服务器集合永远不会改变。

我们为您提供 `client.go` 代码，该代码将每个 RPC 发送到负责该 RPC  键的副本组。如果副本组表示不负责该键，则代码会重试；在这种情况下，客户端代码会向分片控制器请求最新配置并再次尝试。您需要修改 client.go 以支持处理重复的客户端 RPC，这与 kvraft 实验中的做法类似。

完成后，你的代码应能通过除挑战测试外的所有 shardkv 测试：

```
$ cd ~/6.5840/src/shardkv
$ go test
Test (5A): static shards ...
  ... Passed
Test (5A): rejection ...
  ... Passed
Test (5B): join then leave ...
  ... Passed
Test (5B): snapshots, join, and leave ...
labgob warning: Decoding into a non-default variable/field Num may not work
  ... Passed
Test (5B): servers miss configuration changes...
  ... Passed
Test (5B): concurrent puts and configuration changes...
  ... Passed
Test (5B): more concurrent puts and configuration changes...
  ... Passed
Test (5B): concurrent configuration change and restart...
  ... Passed
Test (5B): unreliable 1...
  ... Passed
Test (5B): unreliable 2...
  ... Passed
Test (5B): unreliable 3...
  ... Passed
Test: shard deletion (challenge 1) ...
  ... Passed
Test: unaffected shard access (challenge 2) ...
  ... Passed
Test: partial migration shard access (challenge 2) ...
  ... Passed
PASS
ok  	6.5840/shardkv	173.974s
$
```

您需要让服务器监视配置更改，并在检测到更改时启动分片迁移过程。如果副本组丢失了一个分片，它必须立即停止为该分片中的键提供服务，并开始将该分片的数据迁移到接管所有权的副本组。如果副本组获得了一个分片，它需要等待前一个所有者发送旧分片数据，然后才能接受该分片的请求。

在配置更改期间实现分片迁移。确保副本组中的所有服务器在它们执行的操作序列中的同一时间点进行迁移，以便它们都接受或拒绝并发的客户端请求。你应该在着手后续测试之前，专注于通过第二个测试（“加入然后离开”）。当你通过所有测试直到但不包括 `TestDelete` 时，此任务即告完成。

您的服务器需要定期轮询 shardctrler 以了解新的配置。测试期望您的代码大约每 100 毫秒轮询一次；更频繁是可以的，但频率过低可能会导致问题。

服务器在配置更改期间需要相互发送 RPC 以传输分片。shardctrler 的 `Config` 结构体包含服务器名称，但你需要一个 `labrpc.ClientEnd` 才能发送 RPC。你应该使用传递给 `StartServer()` 的 `make_end()` 函数将服务器名称转换为 `ClientEnd` 。 `shardkv/client.go` 包含执行此操作的代码。

- 逐个按顺序重新配置进程。
- 如果测试失败，请检查 gob 错误（例如“gob: type not registered for interface ...”）。Go 不认为 gob 错误是致命的，尽管它们对实验室来说是致命的。
- 您需要为跨分片移动的客户端请求提供最多一次语义（重复检测）。
- 考虑一下 shardkv 客户端和服务器应如何处理 `ErrWrongGroup` 。如果客户端收到 `ErrWrongGroup` ，是否应更改序列号？服务器在执行 `Get` / `Put` 请求时返回 `ErrWrongGroup` ，是否应更新客户端状态？
- 服务器迁移到新配置后，继续存储不再拥有的分片是可以接受的（尽管在实际系统中这是令人遗憾的）。这可能有助于简化您的服务器实现。
- 当组 G1 在配置变更期间需要从 G2 获取一个分片时，G2 在处理日志条目的过程中何时发送分片给 G1 是否重要？
- 您可以在 RPC 请求或回复中发送整个映射，这可能有助于保持分片传输代码的简洁。
- 如果您的某个 RPC 处理程序在其回复中包含了一个映射（例如键/值映射），而该映射是服务器状态的一部分，您可能会因竞态条件而遇到错误。RPC  系统需要读取该映射以便将其发送给调用者，但它并未持有覆盖该映射的锁。然而，您的服务器可能会在 RPC  系统读取映射的同时继续修改同一映射。解决方法是让 RPC 处理程序在回复中包含映射的副本。
- 如果你在 Raft 日志条目中放入一个映射或切片，而你的键/值服务器随后在 `applyCh` 上看到该条目，并在你的键/值服务器状态中保存了对该映射/切片的引用，那么可能会出现竞态条件。请制作映射/切片的副本，并将副本存储在你的键/值服务器状态中。竞态条件发生在你的键/值服务器修改映射/切片与 Raft 在持久化日志时读取它之间。
- 在配置更改期间，一对组可能需要在它们之间双向移动分片。如果遇到死锁，这可能是其中一个原因。

### 无信用挑战练习

这两个功能对你来说将是必不可少的 要构建一个用于生产环境的系统。

#### 状态垃圾回收

当一个副本组失去对一个分片的所有权时，该副本 组应从其数据库中删除丢失的密钥。 它保留不再拥有的值是浪费的， 并且不再服务于请求。然而，这带来了一些 迁移问题。假设我们有两个组，G1 和 G2， 有一个新的配置 C 将分片 S 从 G1 移动到 G2. 如果 G1 从其数据库中删除 S 中的所有密钥时 过渡到 C 时，G2 如何获取 S 的数据 移动到 C？

确保每个副本组保留旧分片的时间不超过绝对必要的时间。即使像上述 G1 这样的副本组中的所有服务器都崩溃并随后恢复，您的解决方案也必须有效。如果您通过 `TestChallenge1Delete` ，则已完成此挑战。

#### 客户端在配置更改期间的请求

处理配置更改的最简单方法是 禁止所有客户端操作，直到转换完成 完成。虽然概念上简单，但这种方法并不 在生产级系统中可行；它会导致长时间的停顿 为所有客户在机器搬入或搬出时。 继续提供分片服务会更好 不受当前配置更改影响的部分。

修改你的解决方案，使得在配置更改期间，不受影响的分片中的键的客户端操作继续执行。当你通过 `TestChallenge2Unaffected` 时，即完成了此挑战。

虽然上述优化已经不错，但我们还能做得更好。假设某个副本组 G3 在过渡到 C 时，需要从 G1 获取分片 S1，从 G2 获取分片 S2。我们非常希望 G3  一旦接收到必要状态，就能立即开始服务一个分片，即使它仍在等待其他分片。例如，如果 G1 宕机，G3 一旦从 G2  接收到适当的数据，就应该开始服务 S2 的请求，尽管向 C 的过渡尚未完成。

修改你的解决方案，使得副本组一旦能够提供服务，即使配置仍在进行中，也能立即开始服务分片。当你通过 `TestChallenge2Partial` 时，即完成了此挑战。

### Handin 程序

在提交之前，请最后再运行*所有*测试一次。

另外，请注意，你的实验 5 分片服务器、实验 5 分片控制器和实验 4 kvraft 都必须使用相同的 Raft 实现。

在提交之前，请再次检查您的解决方案是否适用于：

```sh
$ go test ./raft
$ go test ./kvraft
$ go test ./shardctrler
$ go test ./shardkv
```