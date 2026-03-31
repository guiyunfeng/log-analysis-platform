# 日志分析告警平台

基于 Loki + Go + Vue3 的日志分析与智能告警系统，支持多维度错误统计、TopN 排行、日志明细下钻、智能异常检测和钉钉告警推送。

## 功能特性

| 功能模块 | 描述 |
|---------|------|
| **错误大盘** | 按 project/service/caller_file/job 多维度展示错误趋势时间序列图（ECharts）|
| **TopN 排行** | 错误最多的服务/调用点排行，支持柱状图 + 表格双展示 |
| **日志明细** | 点击某分组下钻查看具体 content 列表，支持关键字搜索 |
| **告警规则** | CRUD 界面管理 critical/warning/noise 分类规则，支持正则匹配 |
| **告警历史** | 完整告警记录，支持筛选、标记处理，显示环比信息 |
| **异常检测** | 突增检测（当前 vs 历史均值）、定时每分钟分析 |
| **钉钉推送** | critical 立即推送、warning 批量聚合推送、noise 静默 |
| **系统设置** | Loki 地址、钉钉 Webhook、全局告警参数一键配置 |

## 架构说明

```
┌──────────────────────────────────────────────┐
│              前端 (Vue3 + ECharts)            │
│  错误大盘 | TopN排行 | 日志明细               │
│  告警规则 | 告警历史 | 系统设置               │
└──────────────────┬───────────────────────────┘
                   │ HTTP API (/api/*)
┌──────────────────▼───────────────────────────┐
│              后端 (Go + Gin)                  │
│  Loki 查询封装 | 异常检测 | 规则匹配           │
│  钉钉推送 | 定时任务（每分钟）                 │
└───────┬──────────────┬───────────────────────┘
        │              │
   ┌────▼────┐    ┌────▼────┐
   │  Loki   │    │  MySQL  │
   │ (已有)  │    │  8.0    │
   └─────────┘    └─────────┘
```

### 技术栈

- **后端**：Go 1.21 + Gin + GORM
- **前端**：Vue 3 + Vite + ECharts + Element Plus
- **数据库**：MySQL 8.0
- **日志源**：Loki HTTP API
- **部署**：Docker Compose

## 项目结构

```
log-analysis-platform/
├── docker-compose.yaml       # 一键部署配置
├── Dockerfile.backend        # 后端镜像构建
├── Dockerfile.frontend       # 前端镜像构建
├── backend/
│   ├── main.go               # 入口，路由注册，初始化
│   ├── go.mod
│   ├── config/config.go      # 配置加载（环境变量）
│   ├── model/                # 数据模型
│   │   ├── alert_rule.go
│   │   ├── alert_history.go
│   │   └── setting.go
│   ├── handler/              # HTTP 处理器
│   │   ├── dashboard.go
│   │   ├── alert_rule.go
│   │   ├── alert_history.go
│   │   ├── log_detail.go
│   │   └── setting.go
│   ├── service/              # 业务逻辑
│   │   ├── loki.go           # Loki API 查询封装
│   │   ├── analyzer.go       # 异常检测引擎
│   │   ├── alerter.go        # 告警管理
│   │   └── dingtalk.go       # 钉钉推送
│   ├── job/scheduler.go      # 定时任务（每分钟）
│   └── migration/init.sql    # 数据库初始化 + 预置规则
├── frontend/
│   ├── src/
│   │   ├── views/            # 页面组件
│   │   ├── components/       # 公共组件
│   │   ├── api/index.js      # API 请求封装
│   │   └── router/index.js   # 路由配置
│   └── vite.config.js
└── nginx/default.conf        # Nginx 反向代理配置
```

## 快速部署

### 前置要求

- Docker & Docker Compose
- 已部署的 Loki 实例

### 1. 克隆仓库

```bash
git clone https://github.com/guiyunfeng/log-analysis-platform.git
cd log-analysis-platform
```

### 2. 修改配置

编辑 `docker-compose.yaml`，修改以下环境变量：

```yaml
environment:
  # MySQL 连接（如果修改密码，同步修改 mysql 服务的 MYSQL_ROOT_PASSWORD）
  - MYSQL_DSN=root:password@tcp(mysql:3306)/log_analysis?charset=utf8mb4&parseTime=True&loc=Local
  
  # Loki 服务地址（替换为你的实际地址）
  - LOKI_URL=http://172.26.240.15:3100
  
  # 钉钉机器人 Webhook（替换为你的实际 token）
  - DINGTALK_WEBHOOK=https://oapi.dingtalk.com/robot/send?access_token=your_token_here
```

### 3. 启动服务

```bash
docker-compose up -d
```

### 4. 访问平台

- **前端界面**：`http://你的服务器IP:8088`
- **后端 API**：`http://你的服务器IP:8080`
- **数据库**：`localhost:13306`（用于调试）

### 5. 查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看后端日志
docker-compose logs -f backend
```

## 配置说明

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `MYSQL_DSN` | `root:password@tcp(127.0.0.1:3306)/log_analysis?...` | MySQL 连接字符串 |
| `LOKI_URL` | `http://localhost:3100` | Loki 服务地址 |
| `DINGTALK_WEBHOOK` | 空 | 钉钉机器人 Webhook URL |
| `SERVER_PORT` | `8080` | 后端监听端口 |

### 全局告警参数（系统设置页面可调）

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `spike_multiplier` | `10` | 突增倍数阈值，当前5分钟 / 历史均值超过此倍数触发告警 |
| `global_threshold` | `100` | 未匹配规则的服务5分钟错误数超过此值触发 warning |
| `global_time_window` | `300` | 全局默认时间窗口（秒） |
| `global_silence_minutes` | `30` | 同服务+级别告警的最小发送间隔（分钟） |
| `warning_batch_interval` | `5` | Warning 聚合推送间隔（分钟） |

## 预置告警规则

系统启动时自动创建以下预置规则：

| 规则名 | 级别 | 匹配条件 | 阈值 |
|--------|------|----------|------|
| DB扫描错误 | critical | 内容含 `Scan error on column index` | >0次 |
| 连接失败 | critical | 内容含 `connect failed\|connection refused` | >0次 |
| 用户未找到 | warning | 内容含 `not found` | 5分钟>50次 |
| 风控报单错误 | warning | 内容含 `ErrCode:205010` | 5分钟>20次 |
| 未授权请求噪音 | noise | 内容含 `no token present in request` | - |
| 扫描器噪音 | noise | 内容含 `CensysInspect` | - |

## 钉钉告警示例

```
🔴 [CRITICAL] 服务异常告警
━━━━━━━━━━━━━━━━━━━━━
**项目:** ai_quant
**服务:** risk
**调用点:** control/riskgroup.go
**机器:** aliyun-ait0-ai-worker-47-242-124-68-test
━━━━━━━━━━━━━━━━━━━━━
**过去5分钟错误:** 87 次
环比上一小时: ↑ 340%
━━━━━━━━━━━━━━━━━━━━━
**示例报错:**
query1 [SGX_CN2603] quote err: ErrCode:205010...
━━━━━━━━━━━━━━━━━━━━━
⏰ 2026-03-31 15:30:00
```

## API 文档

### Dashboard

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/dashboard/error-trend` | 错误趋势时间序列 |
| GET | `/api/dashboard/error-summary` | 各维度错误汇总 |

### TopN

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/topn/services` | 错误最多的服务 TopN |
| GET | `/api/topn/callers` | 错误最多的调用点 TopN |

### 日志明细

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/logs` | 日志明细列表（支持筛选/分页） |

### 告警规则

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/alert-rules` | 获取规则列表 |
| POST | `/api/alert-rules` | 创建规则 |
| PUT | `/api/alert-rules/:id` | 更新规则 |
| DELETE | `/api/alert-rules/:id` | 删除规则 |
| PUT | `/api/alert-rules/:id/toggle` | 启用/禁用规则 |

### 告警历史

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/alert-history` | 告警历史列表（分页+筛选） |
| PUT | `/api/alert-history/:id/resolve` | 标记为已处理 |

### 系统设置

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/settings` | 获取所有配置 |
| PUT | `/api/settings` | 批量更新配置 |

## 使用说明

### 错误大盘

1. 在顶部工具栏选择时间范围（5分钟/15分钟/1小时/6小时/12小时/24小时/7天）
2. 可按项目/服务/机器筛选
3. 错误趋势图展示各服务的错误速率（次/秒）
4. 下方饼图展示错误在各服务/项目间的分布

### TopN 排行

1. 选择时间范围和展示数量（Top 5/10/20）
2. 切换图表/表格/两者展示模式
3. 点击「下钻查看」跳转到日志明细页，自动带入该服务/调用点的筛选条件

### 日志明细

1. 输入筛选条件（项目、服务、调用点、关键字）
2. 点击任意行查看日志完整详情（timestamp、caller、content、trace、span）
3. 支持分页浏览

### 告警规则管理

1. 点击「新增规则」创建告警规则
2. 规则支持：按项目/服务/调用点/内容关键字（正则）匹配
3. 三种级别：Critical（立即推送）、Warning（聚合推送）、Noise（静默）
4. 可直接在表格中切换规则启用/禁用状态

### 系统设置

1. 配置 Loki 地址后点击「测试连接」验证
2. 配置钉钉 Webhook 后所有告警自动推送
3. 全局告警参数调整后实时生效

## 常见问题

**Q: 图表没有数据怎么排查？**
A: 
1. 检查系统设置中 Loki 地址是否正确
2. 确认 Loki 中有 `logtype="error"` 标签的日志
3. 查看后端日志：`docker-compose logs backend`

**Q: 钉钉收不到消息？**
A:
1. 确认 Webhook URL 正确
2. 钉钉机器人如使用「关键字」安全模式，确认消息中包含该关键字（告警消息包含「告警」字样）
3. 检查是否在静默时间内

**Q: 如何修改数据库密码？**
A: 修改 `docker-compose.yaml` 中 `MYSQL_ROOT_PASSWORD` 和 `MYSQL_DSN` 中的密码，然后 `docker-compose up -d --force-recreate`

## License

MIT
