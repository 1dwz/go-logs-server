# 日志服务器 API 文档

## 概述

日志服务器是一款专为 AutoHotkey 脚本设计的轻量级日志服务器，采用 Go 语言开发，提供 HTTP API 接口与 Web 界面。

**重要提示**：日志存储在内存中，服务重启后日志将被清空。如需持久化存储，请使用导出功能。

## 服务器地址

- 默认监听地址：`http://0.0.0.0:29121`
- Web 界面主页：`http://localhost:29121/`
- Web 详情页：`http://localhost:29121/view?title=脚本名称`
- API 基础路径：`http://localhost:29121/api`

---

## 1. 日志写入接口

与 LogClient.ahk 无缝适配的主要接口。

### 请求

```
POST /log
```

**URL 参数：**

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| title | string | 是 | 日志标题 |
| level | string | 是 | 日志级别：info / warn / error / debug |
| tag | string | 否 | 标签，用于分类 |

**请求体：**

纯文本格式的日志消息内容。

### 响应

**成功响应：**

```
HTTP/1.1 200 OK
Content-Type: text/plain

OK
```

### 示例

```bash
curl -X POST "http://localhost:29121/log?title=脚本1&level=info&tag=init" \
     -d "Application started successfully"
```

---

## 2. 标题列表接口

获取所有日志标题（去重后按 A-Z 排序）。

### 请求

```
GET /api/titles
```

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "titles": [
      {
        "name": "脚本1",
        "count": 10,
        "lastTime": "2024-01-15T10:30:00+08:00",
        "levelDist": {
          "info": 5,
          "warn": 3,
          "error": 2,
          "debug": 0
        }
      },
      {
        "name": "脚本2",
        "count": 5,
        "lastTime": "2024-01-15T09:00:00+08:00",
        "levelDist": {
          "info": 3,
          "warn": 1,
          "error": 0,
          "debug": 1
        }
      },
      {
        "name": "未知",
        "count": 2,
        "lastTime": "2024-01-15T08:00:00+08:00",
        "levelDist": {
          "info": 2,
          "warn": 0,
          "error": 0,
          "debug": 0
        }
      }
    ],
    "total": 3
  }
}
```

### 说明

- `name`: 标题名称，无标题的日志显示为"未知"
- `count`: 该标题下的日志总数
- `lastTime`: 最新日志时间（ISO8601 格式）
- `levelDist`: 各级别日志分布统计

---

## 3. 日志查询接口

查询指定条件的日志列表。

### 请求

```
GET /api/logs
```

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| title | string | 按标题精确筛选 |
| level | string | 按级别筛选：info / warn / error / debug |
| tag | string | 按标签筛选 |
| keyword | string | 搜索日志消息内容 |
| page | int | 页码，从 1 开始（默认 1） |
| pageSize | int | 每页数量，最大 200（默认 50） |

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "logs": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "title": "脚本1",
        "level": "info",
        "tag": "init",
        "message": "Application started successfully",
        "timestamp": "2024-01-15T10:30:00+08:00"
      }
    ],
    "total": 100,
    "page": 1,
    "pageSize": 50
  }
}
```

### 示例

```bash
# 查询指定标题的日志
curl "http://localhost:29121/api/logs?title=脚本1"

# 按级别筛选
curl "http://localhost:29121/api/logs?level=error"

# 按标签筛选（支持多标签，逗号分隔）
curl "http://localhost:29121/api/logs?tag=init,database"

# 分页查询
curl "http://localhost:29121/api/logs?page=2&pageSize=20"
```

---

## 4. 标签列表接口

获取指定标题下的所有标签列表（用于筛选）。

### 请求

```
GET /api/tags
```

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| title | string | 按标题筛选（必需） |

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": ["database", "init", "network"]
}
```

### 示例

```bash
curl "http://localhost:29121/api/tags?title=脚本1"
```

---

## 5. 实时日志流接口

通过 Server-Sent Events (SSE) 实时推送新日志（按条件过滤）。

### 请求

```
GET /api/logs/stream
```

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| title | string | 按标题筛选 |
| level | string | 按级别筛选（默认 info） |

### 响应

```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive

event: log
data: {"id":"xxx","title":"脚本1","level":"info","message":"...","timestamp":"..."}
```

---

## 6. 统计接口

获取日志统计数据。

### 请求

```
GET /api/stats
```

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "todayTotal": 150,
    "byLevel": {
      "info": 100,
      "warn": 30,
      "error": 15,
      "debug": 5
    }
  }
}
```

---

## 7. 导出接口

导出指定条件的日志为 JSON 文件。

### 请求

```
GET /api/export
```

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| title | string | 按标题筛选（必需） |
| level | string | 按级别筛选（可选） |
| tag | string | 按标签筛选（可选） |

### 响应

返回 JSON 格式的日志数组，Content-Disposition 头指定下载文件名。

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "脚本1",
    "level": "info",
    "tag": "init",
    "message": "Application started successfully",
    "timestamp": "2024-01-15T10:30:00+08:00"
  }
]
```

### 示例

```bash
# 导出指定标题的所有日志
curl -o logs.json "http://localhost:29121/api/export?title=脚本1"

# 导出指定标题的 error 级别日志
curl -o errors.json "http://localhost:29121/api/export?title=脚本1&level=error"
```

### 限制

- 每次导出最多返回 10000 条日志
- 超过限制时自动截取最新的 10000 条

---

## 8. Web 界面

### 主页 (`/`)

显示所有日志标题列表，按 A-Z 字母排序（支持中文）。

功能：
- 显示标题卡片，包含日志数量
- 显示各级别日志分布
- 显示最后一条日志时间
- 点击标题进入详情页

### 详情页 (`/view?title=xxx`)

显示指定标题下的日志列表。

功能：
- 自动刷新开关（默认开启，每 3 秒刷新）
- Level 筛选（默认只显示 info）
- Tag 标签按钮筛选（支持多选）
- 导出 JSON 按钮
- 分页显示
- 点击日志查看详情弹窗
- 返回主页按钮

---

## 9. 错误码说明

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| -1 | 通用错误 |

---

## 10. 日志级别

| 级别 | 说明 | Web 显示颜色 |
|------|------|-------------|
| info | 信息日志 | 蓝色 |
| warn | 警告日志 | 琥珀色 |
| error | 错误日志 | 红色 |
| debug | 调试日志 | 绿色 |

---

## 11. 配置文件

配置文件位于项目根目录的 `config.toml`：

```toml
[server]
host = "0.0.0.0"
port = 29121

[storage]
log_dir = "./logs"
max_file_size = 10485760
max_age_days = 7

[buffer]
queue_size = 10000
flush_interval = "1s"
```

### 配置项说明

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| server.host | 0.0.0.0 | 监听地址 |
| server.port | 29121 | 监听端口 |
| buffer.queue_size | 10000 | 内存中最大日志条数 |
| buffer.flush_interval | 1s | 队列刷新间隔 |

### 内存存储说明

- 日志存储在内存中，服务重启后清空
- 当日志数量超过 `queue_size` 时，自动删除最早的 10% 日志
- 建议根据服务器内存调整 `queue_size`
