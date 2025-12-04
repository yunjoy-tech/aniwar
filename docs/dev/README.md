## 快速上手简易教程

1小时快速在win上创建一个开发环境

### 安装go

todo install-g.bat

### 安装daprd

todo install-daprd.bat

### 部署本地开发数据库

1. 修改 db.yml
2. 启动服务
   docker-compose -f db.yml up -d

3. 验证
   docker-compose ps
   应显示 redis 和 mongo 状态为 Up

4. 连接测试
   redis-cli -h 127.0.0.1 -p 16379 -a 123456
   输入 ping → 应返回 PONG

mongosh "mongodb://admin:123456@127.0.0.1:27017/dapr_app?authSource=admin"
输入 db.runCommand({ping: 1}) → 应返回 { ok: 1 }