# MeshChat Server

基于 Go + Chi + GORM + PostgreSQL + Redis + WebSocket + IPFS 的群聊服务端骨架与核心功能实现。服务端使用 libp2p 公钥/签名 challenge 登录，外部接口不会暴露 `peer_id`。

程序接入方可直接参考 [API.md](/mnt/e/code/meshchat-server/docs/API.md)。

`GET /server/info` 可在不登录的情况下读取当前服务器模式。

## 项目结构

```text
cmd/server/main.go
internal/app
internal/auth
internal/config
internal/db
internal/events
internal/ipfs
internal/middleware
internal/model
internal/repo
internal/redisx
internal/service
internal/transport/http
internal/transport/ws
pkg/apperrors
pkg/logx
configs/.env.example
docker-compose/docker-compose.yml
Dockerfile
Makefile
README.md
```

## 快速启动

### 方式一：Docker Compose 一键启动

```bash
cd docker-compose
docker compose up --build -d
```

启动后服务默认暴露：

- API: `http://localhost:8080`
- IPFS Gateway: `http://localhost:8081`

PostgreSQL、Redis、IPFS API 在 Compose 内部网络中互通，不默认映射到宿主机，避免本机端口冲突。

`docker-compose` 目录下的数据都会落在以下本地目录，不使用命名卷：

- `docker-compose/postgres-data`
- `docker-compose/redis-data`
- `docker-compose/ipfs-data`

如果仓库位于 WSL 的 `/mnt/<盘符>/...` 这类 Windows 文件系统路径，`postgres:latest` 可能因为绑定目录不支持容器内 `chmod` 而无法初始化。遇到这种情况时，建议将仓库移动到 WSL 的 Linux 文件系统路径（例如 `~/code/...`）后再执行 Compose，或者为该挂载启用 Linux metadata。

### 方式二：本地运行

1. 准备 PostgreSQL、Redis、IPFS。
2. 参考 [configs/.env.example](/mnt/e/code/meshchat-server/configs/.env.example) 设置环境变量。
3. 启动服务：

```bash
go run ./cmd/server
```

## 配置项

主要环境变量如下：

- `SERVER_MODE`: `public` 或 `restricted`，控制普通用户是否允许创建群
- `SERVER_ADMIN_PEER_IDS`: 服务器管理员 `peer_id` 列表，使用英文逗号分隔
- `DATABASE_URL`: PostgreSQL 连接串
- `REDIS_ADDR`: Redis 地址
- `JWT_SECRET`: JWT 签名密钥
- `AUTH_CHALLENGE_TTL`: 登录 challenge 生存时间
- `IPFS_API_URL`: IPFS API 地址
- `AUTO_MIGRATE`: 启动时自动执行 GORM migration
- `WS_SEND_BUFFER`: WebSocket 单连接发送队列大小

完整示例见 [configs/.env.example](/mnt/e/code/meshchat-server/configs/.env.example)。

## 数据库初始化

服务启动时默认执行 GORM migration，创建：

- `server_users`
- `groups`
- `group_members`
- `group_messages`
- `group_message_edits`
- `files`

同时会执行：

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;
```

如果你不希望启动时迁移，可将 `AUTO_MIGRATE=false`。

## 认证流程

### 1. 获取 challenge

```bash
curl -X POST http://localhost:8080/auth/challenge \
  -H 'Content-Type: application/json' \
  -d '{"peer_id":"12D3KooW..."}'
```

### 2. 使用 libp2p 私钥签名 challenge 后登录

```bash
curl -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{
    "peer_id":"12D3KooW...",
    "challenge_id":"0b7b1c3a-...",
    "signature":"BASE64_SIGNATURE",
    "public_key":"BASE64_MARSHALLED_LIBP2P_PUBLIC_KEY"
  }'
```

登录成功后返回 JWT token。

## WebSocket 使用

握手地址：

```text
GET /ws?token=<jwt>
```

订阅多个群：

```json
{
  "action": "subscribe",
  "group_ids": ["group-uuid-1", "group-uuid-2"]
}
```

取消订阅：

```json
{
  "action": "unsubscribe",
  "group_ids": ["group-uuid-1"]
}
```

服务端推送事件：

- `group.message.created`
- `group.message.edited`
- `group.message.deleted`
- `group.settings.updated`
- `group.member.updated`

## API 示例

### 获取个人资料

```bash
curl http://localhost:8080/me/profile \
  -H "Authorization: Bearer ${TOKEN}"
```

### 创建群组

```bash
curl -X POST http://localhost:8080/groups \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{
    "title":"MeshChat",
    "about":"backend group",
    "member_list_visibility":"hidden",
    "join_mode":"invite_only"
  }'
```

注意：

- `SERVER_MODE=public` 时，普通用户也可以创建群
- `SERVER_MODE=restricted` 时，只允许 `SERVER_ADMIN_PEER_IDS` 中配置的服务器管理员创建群

### 发文本消息

```bash
curl -X POST http://localhost:8080/groups/${GROUP_ID}/messages \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{
    "content_type":"text",
    "payload":{"text":"hello meshchat"}
  }'
```

### 注册文件元数据

```bash
curl -X POST http://localhost:8080/files \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{
    "cid":"bafy...",
    "mime_type":"application/pdf",
    "size":456789,
    "file_name":"spec.pdf"
  }'
```

## 当前实现说明

- `SERVER_MODE=public` 时所有登录用户都可以创建群，`restricted` 时只有服务器管理员可以创建群。
- 服务器管理员可以解散群；服务器管理员和群主都可以设置或取消群管理员。
- 当前群主可以将群转让给本群的其他活跃成员，转让后原群主自动变为管理员。
- 公开群支持 `POST /groups/{group_id}/join` 自助加入，加入后才能在该群发言。
- 群主、群管理员和服务器管理员支持 `POST /groups/{group_id}/members/{user_id}/invite` 邀请用户入群，邀请后目标用户直接变为活跃成员。
- `POST /groups/{group_id}/leave` 支持主动退出群聊，退出后不再拥有该群权限；公开群后续仍可重新加入。
- 管理后台运行在独立端口，默认 `ADMIN_HTTP_ADDR=:8081`，使用 `ADMIN_USERNAME` / `ADMIN_PASSWORD` 登录。
- 浏览器直接访问后台端口根路径即可打开管理页面。
- `peer_id` 仅用于认证和内部存储，不会出现在 HTTP/WS 响应中。
- `forward` 为引用型转发，不复制原消息快照。
- TTL 与 slow mode 都按群当前配置动态生效。
- 文件只保存 CID 与元数据，不保存二进制内容。
- Redis 同时承担 challenge 缓存、Pub/Sub、slow mode、在线状态。
