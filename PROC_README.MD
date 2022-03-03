## 进程信息监控

本仓库是 Node Exporter 的 fork 版本，相对于基础功能，增加了进程信息监控的能力。完全没有改动 Node Exporter 本身的功能，易于跟踪标准版的更新，也和现有的 Node Exporter 配置完全兼容。

进程监控组件主要来源于 [Process Exporter](https://github.com/ncabatoff/process-exporter) ，我们将其核心采集逻辑迁移到了 Node Exporter，以 collector 的形式集成。同时根据业务要求，添加了更多的监控信息。

### 监控项
1. 以下所有监控项都包含了如下前缀：namedprocess_namegroup_
2. 所有监控项至少包含了 label: groupname=xxx 来表示进程名为 xxx

```
示例：

namedprocess_namegroup_netstat{groupname="AliHips",local="10.48.137.158:38848",protocol="tcp",remote="100.100.167.125:443"} 1

前缀：namedprocess_namegroup_
监控项名称：netstat（每一条 netstat 监控代表该进程的某一个打开的 tcp/udp 链接）
标签：
  groupname="AliHips" // 进程名为 AliHips
  protocol="tcp" // tcp 连接
  local="10.48.137.158:38848" // 本地 ip:port
  remote="100.100.167.125:443" // 远端 ip:port
值：
  1 代表 TCP_ESTABLISHED （其他数字解释见下文详情）
```

##### pid gauge
进程的 pid 信息，gauge 值为 pid 号

##### effective_username gauge
进程的 effective username，gauge 值无意义，可忽略

##### netstat gauge
进程所创建的 tcp/udp 连接信息。

标签：
- protocol: [tcp/udp]
- local: local ip:port
- remote: remote ip:port

gauge 值：（详见：[net/tcp_states.h](https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/include/net/tcp_states.h) ）
- TCP_ESTABLISHED  = 01
- TCP_SYN_SENT     = 02
- TCP_SYN_RECV     = 03
- TCP_FIN_WAIT1    = 04
- TCP_FIN_WAIT2    = 05
- TCP_TIME_WAIT    = 06
- TCP_CLOSE        = 07
- TCP_CLOSE_WAIT   = 08
- TCP_LAST_ACK     = 09
- TCP_LISTEN       = 0A
- TCP_CLOSING      = 0B
- TCP_NEW_SYN_RECV = 0C
- TCP_MAX_STATES   = 0D

##### num_procs gauge
相同进程名存在的数量

##### cpu_seconds_total counter
进程消耗的 cpu 总时间
/proc/stat->utime/stime

##### read_bytes_total counter
/proc/io->read_bytes

##### write_bytes_total counter
/proc/io->write_bytes

##### major_page_faults_total counter
/proc/stat->majflt

##### minor_page_faults_total counter
/proc/stat->minflt

##### context_switches_total counter
标签：
- ctxswitchtype=voluntary
  - /proc/status->voluntary_ctxt_switches
- ctxswitchtype=nonvoluntary
  - /proc/status->nonvoluntary_ctxt_switches

##### memory_bytes gauge
标签：
- memtype=resident
  - /proc/stat->rss
- memtype=virtual
  - /proc/stat->vsize
- memtype=swapped
    - /proc/status->VmSwap
- memtype=proportionalResident
    - /proc/smaps->Pss
- memtype=swapped
    - /proc/smaps->SwapPss

##### open_filedesc gauge
进程打开的fd 总数

##### worst_fd_ratio gauge
/proc/limits

##### oldest_start_time_seconds gauge
/proc/stat->starttime

##### num_threads gauge
进程打开的线程总数

##### states gauge
进程打开的线程，线程每种状态的总数

标签：
- state=Running
- state=Sleeping
- state=Waiting
- state=Zombie
- state=Other

### 开发/发布
与进程监控相关的 go 源文件：
1. collector/process_detail_linux.go 与 Node Exporter 交互的入口
2. collector/process_detail/*.go 实际的进程监控逻辑

编译：
1. 由上述入口文件可知，进程监控只会在 GOOS=linux 时编译，其他平台自动忽略
2. 设置 env GOOS=linux 后，执行 make build 命令即可

追踪 Node Exporter 的新版本：
1. 在当前 fork 仓库下，创建一个新的名为 upstream 的 remote，指向 node exporter 原始仓库
2. 执行 `git fetch upstream` 拉取 upstream 的最新更新
3. 执行 `git merge upstream/master` 合并最新更新 (也可以用 git rebase)

采用 patch 的形式将修改合入最新版本（目前的修改已经打好放在 patch.zip 中）：
1. 将修改打 patch：`git format-patch [commit-id] -o ./patch`
2. 在新的仓库中应用这些 patch 文件：`git am ./patch/*.patch`