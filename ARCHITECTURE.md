# Aniwar 游戏服务器架构文档

## 项目概述

**项目名称**: Aniwar 游戏服务器
**技术栈**: Go 1.22 + Dapr + MongoDB + Redis
**架构模式**: 微服务架构
**游戏类型**: 卡牌/策略类手游
**底层框架**: Musae (../musae)

---

## 目录结构

```
aniwar/
├── src/                    # 源代码目录
│   ├── actorserver/        # Actor 模式微服务（游戏逻辑核心）
│   ├── billserver/         # 支付/计费服务
│   ├── gateserver/         # 网关服务（客户端入口）
│   ├── guideserver/        # 新手引导服务
│   ├── idipserver/         # 外部服务集成（奖励、活动）
│   ├── lobbyserver/        # 大厅服务
│   ├── loginserver/        # 登录认证服务
│   ├── common/             # 共享组件
│   └── meta/               # 游戏元数据和配置
├── script/                 # 部署和维护脚本
├── tools/                  # 开发工具
├── docs/                   # 文档
├── proto/                  # Protocol Buffer 定义
└── output/                 # 构建产物
```

---

## 核心服务模块

| 服务 | 功能 | 关键职责 |
|------|------|----------|
| **Login Server** | 用户登录认证 | 多渠道登录（TapTap、Lilith、快豹）、会话管理 |
| **Gate Server** | 客户端连接网关 | 消息路由、协议转换、限流防护、DDoS 防护 |
| **Actor Server** | 游戏核心逻辑 | 用户Actor、房间Actor、联盟Actor、游戏状态管理 |
| **Lobby Server** | 游戏大厅 | 匹配系统、房间管理、玩家连接 |
| **Guide Server** | 新手教程系统 | 新玩家引导流程管理 |
| **Bill Server** | 支付处理 | 内购处理、计费逻辑 |
| **IDIP Server** | 外部服务对接 | 奖励发放、活动管理、GM 工具对接 |

---

## 游戏系统

- 🃏 **卡牌系统**: 卡牌收集、升级、战斗机制
- 👥 **联盟系统**: 公会/氏族功能、联盟战
- ✉️ **邮件系统**: 游戏内消息、系统通知
- 🛒 **商店系统**: 虚拟商品交易、商城
- 🎉 **活动系统**: 限时活动、促销活动
- 📋 **任务系统**: 任务追踪、成就系统

---

## 技术架构

### 1. 微服务模式
- 独立部署和扩展
- gRPC 服务间通信
- Dapr 服务管理和状态管理

### 2. Actor 模型
- 用户和游戏实体作为 Actor
- 隔离的状态管理
- 异步消息传递

### 3. 数据存储
- **MongoDB**: 持久化游戏数据
- **Redis**: 缓存和实时数据
- **Consul**: 服务发现

### 4. 配置管理
- **Apollo 配置中心**: 动态配置管理
- 计划迁移到 **Nacos**

### 5. 监控和日志
- Prometheus 指标采集
- 结构化日志
- Grafana 可视化

---

## 配置文件

| 文件 | 用途 |
|------|------|
| `aniwar-server.yaml.json` | Apollo 配置中心输出（服务器端点、数据库连接、API 密钥） |
| `Dockerfile` | 容器化部署 |
| `go.mod/go.sum` | Go 依赖管理 |
| `Makefile/make.bat` | 构建自动化 |
| `start.bat/stop.bat` | Windows 服务管理 |

---

## 文档

| 文档 | 内容 |
|------|------|
| `README.md` | 基本设置说明 |
| `docs/debug.md` | 调试流程 |
| `docs/pprof.md` | 性能分析指南 |
| `docs/dev/` | 开发文档 |
| `docs/jenkins/` | CI/CD 文档 |
| `docs/grafana/` | 监控面板配置 |

---

## 待办事项 (根据 todo.md)

- [ ] 聊天系统从 Elasticsearch 迁移到 Redis
- [ ] 实现战斗系统 demo
- [ ] 改进错误日志（堆栈跟踪）
- [ ] 配置中心迁移到 Nacos
- [ ] 代码简化和清理
- [ ] 数据库抽象层改进

---

## 安全特性

- RSA 加密敏感数据
- JWT 认证
- 限流和 DDoS 防护
- API 端点 IP 白名单

---

## 构建和部署

**支持的平台**: Windows / Linux
**部署方式**:
- 容器化部署 (Docker)
- 自动化 CI/CD (Jenkins)
- 配置中心动态配置

---

## 依赖框架

本项目基于 **Musae 游戏框架** 构建，详见: `../musae/ARCHITECTURE.md`
