# StudyRoom gRPC 分布式系统 - 完整项目演示

本指南将带您完整地过一遍整个项目，从清理环境到运行测试。

---

## 📋 目录

1. [环境清理](#1-环境清理)
2. [项目结构概览](#2-项目结构概览)
3. [启动5节点集群](#3-启动5节点集群)
4. [验证集群运行](#4-验证集群运行)
5. [运行测试套件](#5-运行测试套件)
6. [代码实现说明](#6-代码实现说明)
7. [关键功能演示](#7-关键功能演示)

---

## 1. 环境清理

### 1.1 停止所有容器

```bash
# 停止gRPC集群
docker compose -f docker-compose-grpc.yml down

# 停止REST API版本（如果运行中）
docker compose down

# 停止测试容器
docker compose -f docker-compose.test.yml down
```

### 1.2 清理未使用的资源

```bash
# 清理未使用的容器
docker container prune -f

# 清理未使用的网络
docker network prune -f

# 查看清理结果
docker ps -a
docker network ls
```

### 1.3 检查端口占用

```bash
# 检查Redis端口(6379)
sudo lsof -i :6379 || echo "6379端口可用"

# 检查MongoDB端口(27017)
sudo lsof -i :27017 || echo "27017端口可用"

# 检查gRPC端口(50051-50055)
for port in 50051 50052 50053 50054 50055; do
  sudo lsof -i :$port || echo "$port端口可用"
done
```

---

## 2. 项目结构概览

```
studyroom_booking/
├── api/
│   └── proto/                    # Protocol Buffer定义文件
│       ├── studyroom.proto      # 业务服务定义（Auth, Booking, Search, Admin）
│       ├── raft.proto           # Raft共识协议定义
│       └── twopc.proto          # 2PC事务协议定义
│
├── cmd/
│   ├── server/                  # REST API服务器（原始版本）
│   └── server-grpc/            # gRPC分布式服务器（main.go）
│
├── internal/
│   ├── raft/                    # Raft共识算法实现
│   │   ├── node.go             # 核心Raft节点逻辑
│   │   ├── client.go           # Raft gRPC客户端
│   │   └── server.go           # Raft gRPC服务器
│   │
│   ├── twopc/                   # 2PC分布式事务实现
│   │   ├── coordinator.go      # 2PC协调者（投票和决策阶段）
│   │   ├── participant.go      # 2PC参与者
│   │   └── server.go           # 2PC gRPC服务器
│   │
│   └── grpc/
│       └── handler/             # gRPC业务处理器
│           ├── auth_handler.go
│           ├── booking_handler.go  # 集成2PC和请求转发
│           ├── search_handler.go
│           └── admin_handler.go
│
├── test/                        # 测试套件
│   ├── twopc_test.go           # 2PC测试（3个测试）
│   ├── raft_test.go            # Raft测试（5个测试）
│   ├── integration_test.go     # 集成测试
│   └── docker_test.sh          # Docker测试脚本
│
├── docker-compose-grpc.yml     # 5节点集群配置
├── docker-compose.test.yml      # 测试环境配置
├── Dockerfile-grpc              # gRPC服务器镜像
└── Dockerfile.test              # 测试环境镜像
```

---

## 3. 启动5节点集群

### 3.1 构建并启动

```bash
# 构建并启动所有服务（5个节点 + MongoDB + Redis）
docker compose -f docker-compose-grpc.yml up -d --build

# 这个命令会：
# 1. 构建gRPC服务器镜像（包含proto文件生成）
# 2. 启动MongoDB容器
# 3. 启动Redis容器（无端口映射，仅内部访问）
# 4. 启动5个gRPC节点（node1-node5）
```

### 3.2 检查启动状态

```bash
# 查看所有容器状态
docker compose -f docker-compose-grpc.yml ps

# 预期输出：7个容器全部运行
# - studyroom-mongo
# - studyroom-redis
# - studyroom_booking-app-node1-1
# - studyroom_booking-app-node2-1
# - studyroom_booking-app-node3-1
# - studyroom_booking-app-node4-1
# - studyroom_booking-app-node5-1
```

### 3.3 查看启动日志

```bash
# 查看node1的启动日志
docker logs studyroom_booking-app-node1-1

# 应该看到：
# - "gRPC server listening on :50051"
# - "Raft node node1 started"
# - "[Raft node1] Starting node"
# - "[Raft node1] Election timeout, starting election"
```

---

## 4. 验证集群运行

### 4.1 查看Raft选举过程

```bash
# 实时查看node1日志（观察选举）
docker logs studyroom_booking-app-node1-1 -f

# 在另一个终端查看其他节点
docker logs studyroom_booking-app-node2-1 | grep -E "(Node|Leader|Follower)" | head -10
docker logs studyroom_booking-app-node3-1 | grep -E "(Node|Leader|Follower)" | head -10
```

### 4.2 查找Leader节点

```bash
# 检查哪个节点是Leader
for i in 1 2 3 4 5; do
  echo "=== Node $i ==="
  docker logs studyroom_booking-app-node${i}-1 2>&1 | grep -i "leader\|becoming leader" | head -3
done
```

### 4.3 查看Raft RPC日志格式

```bash
# 查看Raft RPC调用日志（客户端格式）
docker logs studyroom_booking-app-node1-1 2>&1 | grep "sends RPC" | head -5

# 查看Raft RPC接收日志（服务器格式）
docker logs studyroom_booking-app-node2-1 2>&1 | grep "runs RPC" | head -5
```

**预期日志格式**：
- 客户端：`Node node1 sends RPC RequestVote to Node node2`
- 服务器：`Node node2 runs RPC RequestVote called by Node node1`

---

## 5. 运行测试套件

### 5.1 运行所有测试

```bash
# 运行完整的测试套件
docker compose -f docker-compose.test.yml up --build

# 这个命令会：
# 1. 生成Protocol Buffer代码
# 2. 运行所有2PC测试（3个）
# 3. 运行所有Raft测试（5个）
# 4. 显示测试结果摘要
```

### 5.2 查看测试输出

测试输出会显示：

**2PC测试日志格式**：
```
Phase Voting of Node test-participant receives RPC vote-request from Phase Voting of Node node1
Phase Voting of Node test-participant sends RPC vote-commit to Phase Voting of Node node1
```

**Raft测试日志格式**：
```
Node node1 sends RPC RequestVote to Node node2
Node node2 runs RPC RequestVote called by Node node1
```

### 5.3 测试结果

**预期结果**：8个测试全部通过
- ✅ Test2PCBasicCommit
- ✅ Test2PCAbortOnPrepareFailure
- ✅ Test2PCConcurrentTransactions
- ✅ TestRaftLeaderElection
- ✅ TestRaftLeaderTimeout
- ✅ TestRaftLogReplication
- ✅ TestRaftNewNodeJoin
- ✅ TestRaftSplitBrainPrevention

---

## 6. 代码实现说明

### 6.1 2PC实现（Q1 & Q2）

#### 投票阶段（Q1）- 代码位置

1. **Proto定义**: `api/proto/twopc.proto`
   - `Prepare` RPC = vote-request
   - `PrepareResponse.can_commit=true` = vote-commit
   - `PrepareResponse.can_commit=false` = vote-abort

2. **协调者发送vote-request**: `internal/twopc/coordinator.go:275`
   ```go
   fmt.Printf("Phase Voting of Node %s sends RPC vote-request to Phase Voting of Node %s\n", 
              coordinatorNodeID, participant.NodeID)
   ```

3. **参与者接收vote-request**: `internal/twopc/participant.go:64`
   ```go
   fmt.Printf("Phase Voting of Node %s receives RPC vote-request from Phase Voting of Node %s\n", 
              p.nodeID, coordinatorNodeID)
   ```

4. **参与者发送vote-commit/abort**: `internal/twopc/participant.go:114` 或 `103`
   ```go
   fmt.Printf("Phase Voting of Node %s sends RPC vote-commit to Phase Voting of Node %s\n", ...)
   // 或
   fmt.Printf("Phase Voting of Node %s sends RPC vote-abort to Phase Voting of Node %s\n", ...)
   ```

#### 决策阶段（Q2）- 代码位置

1. **协调者发送global-commit**: `internal/twopc/coordinator.go:302`
   ```go
   fmt.Printf("Phase Decision of Node %s sends RPC global-commit to Phase Decision of Node %s\n", ...)
   ```

2. **协调者发送global-abort**: `internal/twopc/coordinator.go:328`
   ```go
   fmt.Printf("Phase Decision of Node %s sends RPC global-abort to Phase Decision of Node %s\n", ...)
   ```

3. **参与者接收global-commit**: `internal/twopc/participant.go:119`
   ```go
   fmt.Printf("Phase Decision of Node %s receives RPC global-commit from Phase Decision of Node %s\n", ...)
   ```

4. **参与者接收global-abort**: `internal/twopc/participant.go:163`
   ```go
   fmt.Printf("Phase Decision of Node %s receives RPC global-abort from Phase Decision of Node %s\n", ...)
   ```

### 6.2 Raft实现（Q3 & Q4）

#### Leader选举（Q3）- 代码位置

1. **超时设置**: `internal/raft/node.go:80-81`
   ```go
   heartbeatInterval: 1 * time.Second,        // Q3要求：1秒
   electionTimeout:  1500 * time.Millisecond, // Q3要求：基础1.5秒
   ```

2. **随机化选举超时**: `internal/raft/node.go:192-196`
   ```go
   // 随机超时：1.5-3秒
   randomOffset := time.Duration(rand.Intn(1500)) * time.Millisecond
   timeout := n.electionTimeout + randomOffset
   ```

3. **选举逻辑**: `internal/raft/node.go:218`
   - `startElection()`: 成为Candidate，请求投票

4. **客户端发送RequestVote**: `internal/raft/client.go:37`
   ```go
   fmt.Printf("Node %s sends RPC RequestVote to Node %s\n", candidateID, targetNodeID)
   ```

5. **服务器接收RequestVote**: `internal/raft/server.go:21`
   ```go
   fmt.Printf("Node %s runs RPC RequestVote called by Node %s\n", 
              s.node.GetID(), req.CandidateId)
   ```

#### 日志复制（Q4）- 代码位置

1. **Leader追加命令**: `internal/raft/node.go:150`
   - `AppendCommand()`: 只有Leader可以追加

2. **Leader发送AppendEntries**: `internal/raft/client.go:57`
   ```go
   fmt.Printf("Node %s sends RPC AppendEntries to Node %s\n", leaderID, targetNodeID)
   ```

3. **Follower接收AppendEntries**: `internal/raft/server.go:36`
   ```go
   fmt.Printf("Node %s runs RPC AppendEntries called by Node %s\n", 
              s.node.GetID(), req.LeaderId)
   ```

4. **客户端请求转发**: `internal/grpc/handler/booking_handler.go:34`
   ```go
   if h.raftNode != nil && !h.raftNode.IsLeader() {
       return h.forwardToLeader(ctx, ...)  // Q4要求：转发到Leader
   }
   ```

---

## 7. 关键功能演示

### 7.1 演示Raft选举

```bash
# 1. 查看所有节点的状态
for i in 1 2 3 4 5; do
  echo "Node $i:"
  docker logs studyroom_booking-app-node${i}-1 2>&1 | grep -E "(Leader|Follower|Candidate|election)" | tail -3
done

# 2. 观察选举过程
docker logs studyroom_booking-app-node1-1 -f | grep -E "(election|Leader|RequestVote)"
```

### 7.2 演示2PC事务

```bash
# 查看2PC日志（如果有事务发生）
docker logs studyroom_booking-app-node1-1 2>&1 | grep "Phase" | head -10
```

### 7.3 演示日志格式

```bash
# Raft日志格式
echo "=== Raft客户端日志 ==="
docker logs studyroom_booking-app-node1-1 2>&1 | grep "sends RPC" | head -3

echo "=== Raft服务器日志 ==="
docker logs studyroom_booking-app-node2-1 2>&1 | grep "runs RPC" | head -3

# 2PC日志格式（在测试中查看）
echo "=== 2PC日志（运行测试查看） ==="
docker compose -f docker-compose.test.yml up --build 2>&1 | grep "Phase" | head -5
```

---

## 8. 停止和清理

```bash
# 停止5节点集群
docker compose -f docker-compose-grpc.yml down

# 完全清理（包括数据卷）
docker compose -f docker-compose-grpc.yml down -v

# 清理所有未使用的资源
docker system prune -f
```

---

## 📝 快速参考命令

```bash
# 启动集群
docker compose -f docker-compose-grpc.yml up -d --build

# 查看状态
docker compose -f docker-compose-grpc.yml ps

# 查看日志
docker logs studyroom_booking-app-node1-1 -f

# 运行测试
docker compose -f docker-compose.test.yml up --build

# 停止集群
docker compose -f docker-compose-grpc.yml down
```

---

**完成！** 现在您已经完整地过了一遍整个项目。

