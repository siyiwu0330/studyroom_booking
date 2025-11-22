#!/bin/bash
# GitHub上传脚本

echo "=== 开始上传项目到GitHub ==="
echo ""

# 1. 添加所有文件
echo "步骤1: 添加所有文件..."
git add .

# 2. 提交更改
echo ""
echo "步骤2: 提交更改..."
git commit -m "feat: Add distributed gRPC version with Raft and 2PC

- Implement 2PC distributed transactions (Q1 & Q2)
  - Voting phase: vote-request, vote-commit, vote-abort
  - Decision phase: global-commit, global-abort
  - Proper log formatting as required

- Implement Raft consensus algorithm (Q3 & Q4)
  - Leader election with 1s heartbeat, 1.5-3s random election timeout
  - Log replication with client request forwarding
  - Proper log formatting as required

- Deploy 5-node cluster as required by assignment
- Add comprehensive test suite (8 tests, all passing)
- Add documentation (README, TEST_REPORT, DEMO_GUIDE)
- Dockerize all components for easy deployment"

# 3. 确保远程仓库正确
echo ""
echo "步骤3: 配置远程仓库..."
git remote set-url origin https://github.com/siyiwu0330/studyroom_booking.git
git branch -M main

# 4. 推送到GitHub
echo ""
echo "步骤4: 推送到GitHub..."
echo "请确保您有推送权限..."
git push -u origin main

echo ""
echo "=== 上传完成 ==="
echo "访问 https://github.com/siyiwu0330/studyroom_booking 查看项目"
