下面给你一份 **可直接发给 Cursor / Codex 的严格 Codegen Prompt**。
目标是让 AI 按你确定的技术栈，直接生成一个**可运行的群聊服务器后端骨架与核心功能**。

你可以把下面整段原样发给 AI。

---

# Cursor / Codex 严格 Codegen Prompt

你现在要实现一个**生产可扩展的群聊服务器后端**，使用以下固定技术栈，不允许擅自替换：

* Go 1.26+
* Chi
* GORM
* PostgreSQL
* Redis
* WebSocket
* IPFS

这是一个**服务器型群聊系统**，不是端到端加密 IM，也不是联邦协议实现。
用户用 **libp2p 私钥签名 challenge 登录**，服务端校验身份后发放 session/token。
在群聊消息和群成员 API 中，**绝不能暴露用户的 peer_id**。

请直接生成一个**可开发、可运行、结构清晰、便于继续扩展**的后端项目，要求如下。

---

## 一、总体目标

实现一个群聊服务器，支持：

1. 服务器级用户资料
2. 一个用户可加入多个群
3. 群支持多管理员
4. 群支持隐藏成员列表
5. 群支持权限系统
6. 群消息支持：

   * text
   * image
   * video
   * voice
   * file
7. 群消息支持 reply 指定消息
8. 群消息支持 forward，采用**引用型转发**，类似推特转发
9. forward 只能引用**同一服务器**内的消息
10. 用户可以永久编辑自己的消息
11. 用户可在群设置允许范围内撤回自己的消息
12. 管理员可以删除低权限用户的消息
13. 群可配置消息自动删除 TTL
14. 群可配置发言冷却时间 slow mode
15. 所有图片/视频/语音/文件都带 **IPFS CID**
16. 客户端可通过 CID 自行走 IPFS 获取文件
17. WebSocket 支持实时消息推送
18. Redis 用于 Pub/Sub、冷却时间、在线状态和短期缓存

---

## 二、强约束

必须遵守以下约束：

1. **不要暴露 peer_id**

   * `peer_id` 只用于登录、验签、内部审计
   * 群消息接口、成员列表接口、WebSocket 推送中都不能返回 `peer_id`

2. **转发必须是引用型**

   * 不复制原消息内容
   * forward 消息引用服务器内已有消息
   * 原消息编辑后，forward 展示同步变化
   * 原消息删除或过期后，forward 显示“原消息已删除/已过期”

3. **reply 和 forward 互斥**

   * 一条消息不能同时 reply 和 forward

4. **编辑永久允许**

   * 发送者可以永久编辑自己的消息
   * 但只能编辑消息内容 payload
   * 不能改 sender、content_type、reply_to、forward_from

5. **自动删除 TTL 是群级动态策略**

   * 修改后对旧消息也生效
   * 不要把固定 expire_at 写死为唯一依据
   * 消息可见性按群当前 TTL + 消息 created_at 动态计算

6. **slow mode 是群级动态策略**

   * 限制同一用户在同一群的发言频率
   * owner 或有豁免权限的管理员可跳过

7. **文件内容只存 CID 和元数据**

   * 不要把二进制文件存数据库
   * 数据库存文件元数据与 CID

8. **数据库使用 PostgreSQL**

   * GORM 用于模型和事务
   * 热点查询可以用 Raw SQL
   * 不要把所有东西都硬塞进 GORM 自动关联

9. **项目必须可运行**

   * 提供完整目录结构
   * 提供配置文件示例
   * 提供 SQL/GORM migration
   * 提供 Docker Compose
   * 可以一键启动本地开发环境

---

## 三、项目结构要求

生成一个清晰的 Go 项目结构，例如：

```text
cmd/server/main.go
internal/app
internal/config
internal/db
internal/model
internal/repo
internal/service
internal/auth
internal/ipfs
internal/redisx
internal/events
internal/middleware
internal/transport/http
internal/transport/ws
pkg/
configs/
migrations/
docker-compose.yml
Makefile
README.md
```

要求：

1. 分层明确
2. handler 不直接写复杂业务逻辑
3. repo 负责 DB 读写
4. service 负责业务规则
5. ws 单独做 hub / connection manager
6. Redis 封装统一 helper
7. IPFS 封装统一 client/interface

---

## 四、数据库模型要求

请按以下业务模型实现 GORM 模型、迁移和必要索引。

### 1. server_users

字段至少包括：

* id
* peer_id（唯一，内部使用）
* public_key
* username
* display_name
* avatar_cid
* bio
* profile_version
* status
* created_at
* updated_at

### 2. groups

字段至少包括：

* id
* group_id（uuid）
* title
* about
* avatar_cid
* owner_user_id
* member_list_visibility
* join_mode
* default_permissions
* message_ttl_seconds
* message_retract_seconds
* message_cooldown_seconds
* last_message_seq
* settings_version
* status
* created_at
* updated_at

### 3. group_members

字段至少包括：

* id
* group_id
* user_id
* role
* status
* title
* joined_at
* muted_until
* permissions_allow
* permissions_deny
* created_at
* updated_at

唯一约束：

* `(group_id, user_id)`

### 4. group_messages

字段至少包括：

* id
* group_id
* message_id
* sender_user_id
* seq
* content_type
* payload_json
* reply_to_message_id
* forward_from_message_id
* status
* edit_count
* last_edited_at
* last_edited_by_user_id
* deleted_at
* deleted_by_user_id
* delete_reason
* signature
* created_at
* updated_at

唯一约束：

* `(group_id, message_id)`
* `(group_id, seq)`

索引至少包括：

* `(group_id, created_at desc)`
* `(group_id, sender_user_id, created_at desc)`
* `(group_id, reply_to_message_id)`
* `(forward_from_message_id)`

### 5. group_message_edits

字段至少包括：

* id
* group_id
* message_id
* editor_user_id
* old_payload_json
* new_payload_json
* created_at

### 6. files

字段至少包括：

* id
* cid（唯一）
* mime_type
* size
* width
* height
* duration_seconds
* file_name
* thumbnail_cid
* created_by_user_id
* created_at

---

## 五、枚举与权限要求

请定义清晰的常量，不要把魔法数字散落在代码里。

### role

* owner
* admin
* member
* restricted

### member status

* active
* left
* kicked
* banned

### message content type

* text
* image
* video
* voice
* file
* forward

### message status

* normal
* deleted

### delete reason

* self_retracted
* admin_removed

### 权限位

至少实现：

* PERM_SEND_TEXT
* PERM_SEND_IMAGE
* PERM_SEND_VIDEO
* PERM_SEND_VOICE
* PERM_SEND_FILE
* PERM_REPLY
* PERM_FORWARD
* PERM_EDIT_OWN_MESSAGES
* PERM_DELETE_OWN_MESSAGES
* PERM_DELETE_MESSAGES
* PERM_VIEW_MEMBERS
* PERM_BYPASS_SLOWMODE
* PERM_MANAGE_MESSAGE_POLICY
* PERM_EDIT_GROUP_INFO
* PERM_MUTE_MEMBERS
* PERM_BAN_MEMBERS
* PERM_SET_MEMBER_PERMISSIONS

并实现：

```text
effective_permissions =
(role_base_permissions OR permissions_allow) AND (~permissions_deny)
```

owner 直接视为最高权限。

---

## 六、消息 payload 要求

请为不同 content_type 定义结构体，并做严格校验。

### text

```json
{
  "text": "hello"
}
```

### image

```json
{
  "cid": "bafy...",
  "mime_type": "image/jpeg",
  "size": 123456,
  "width": 1280,
  "height": 720,
  "caption": "xxx",
  "thumbnail_cid": "bafy..."
}
```

### video

```json
{
  "cid": "bafy...",
  "mime_type": "video/mp4",
  "size": 12345678,
  "width": 1920,
  "height": 1080,
  "duration": 35,
  "caption": "xxx",
  "thumbnail_cid": "bafy..."
}
```

### voice

```json
{
  "cid": "bafy...",
  "mime_type": "audio/ogg",
  "size": 45678,
  "duration": 12,
  "waveform": "base64..."
}
```

### file

```json
{
  "cid": "bafy...",
  "mime_type": "application/pdf",
  "size": 456789,
  "file_name": "spec.pdf",
  "caption": "xxx"
}
```

### forward

```json
{
  "comment": "optional text"
}
```

---

## 七、认证要求

实现 challenge 登录流程：

### 1. 获取 challenge

`POST /auth/challenge`

输入：

* peer_id

输出：

* challenge_id
* challenge
* expires_at

### 2. 提交签名登录

`POST /auth/login`

输入：

* peer_id
* challenge_id
* signature
* public_key

要求：

* 校验 challenge 未过期
* 校验 public_key 与 peer_id 对应
* 校验签名正确
* 成功后查找或创建 server_user
* 返回 token

可以先把 libp2p 签名校验设计成接口，主流程完整，必要时放一个明确 TODO，但整体结构必须可接入真实校验。

---

## 八、HTTP API 要求

至少实现以下 API。

### 用户资料

* `GET /me/profile`
* `PATCH /me/profile`

### 群

* `POST /groups`
* `GET /groups/{group_id}`
* `PATCH /groups/{group_id}`
* `PATCH /groups/{group_id}/message-policy`

### 成员

* `GET /groups/{group_id}/members`
* `PATCH /groups/{group_id}/members/{user_id}/permissions`
* `POST /groups/{group_id}/members/{user_id}/mute`
* `POST /groups/{group_id}/members/{user_id}/ban`

### 消息

* `GET /groups/{group_id}/messages`
* `POST /groups/{group_id}/messages`
* `PATCH /groups/{group_id}/messages/{message_id}`
* `POST /groups/{group_id}/messages/{message_id}/retract`
* `POST /groups/{group_id}/messages/{message_id}/delete`

### 文件

* `POST /files`

---

## 九、消息规则要求

### 发送消息

必须校验：

1. 用户已登录
2. 群成员状态有效
3. 未被 ban
4. 未被 mute
5. 有对应内容类型权限
6. 未触发 slow mode
7. reply / forward 合法
8. payload 合法
9. 文件类消息的 CID 格式合法

### reply

* 只能 reply 同群消息
* reply 目标消息不存在则拒绝
* reply 目标已删除/过期时返回引用失效信息

### forward

* 只能引用同服务器内消息
* 可以跨群，但原消息必须对当前用户可见
* `forward_from_message_id` 必须合法
* 原消息编辑后，forward 展示实时反映变化
* 原消息删除或过期后，forward 仍存在，但显示“原消息已删除/已过期”

### 编辑

* 永久允许编辑自己的消息
* 仅发送者本人可编辑
* 消息未删除
* 仅允许修改 payload 的可编辑部分
* 不允许修改：

  * sender
  * content_type
  * reply_to_message_id
  * forward_from_message_id
* 必须写入 `group_message_edits`

### 撤回

* 用户只能撤回自己的消息
* 受 `message_retract_seconds` 约束
* 更新消息状态为 deleted

### 管理员删除

* 管理员可删除低权限用户消息
* 不能删除 owner 消息
* 低级管理员不能删除更高级管理员消息

### TTL

消息可见性按当前群 TTL 动态计算：

```text
if ttl == 0 => visible
else visible if created_at >= now - ttl
```

### slow mode

* 作用于 `(group_id, user_id)`
* owner 和具备 `PERM_BYPASS_SLOWMODE` 的管理员豁免
* Redis 保存冷却键
* 返回 `retry_after_seconds`

---

## 十、WebSocket 要求

实现一个可运行的 WebSocket 模块。

### 握手

* `GET /ws`
* 通过 token 认证

### 订阅

客户端可订阅多个 group room

### 服务端推送事件至少包括：

* `group.message.created`
* `group.message.edited`
* `group.message.deleted`
* `group.settings.updated`
* `group.member.updated`

### 架构要求

* 本机内存 hub
* Redis Pub/Sub 跨实例广播
* 单连接发送队列
* 背压时断开慢连接，防止内存无限增长

---

## 十一、Redis 要求

必须实现统一 redis key helper。

至少覆盖：

### 1. slow mode

键示例：

```text
chat:cooldown:{groupID}:{userID}
```

### 2. 在线状态

键示例：

```text
chat:online:user:{userID}
chat:online:group:{groupID}:{userID}
```

### 3. Pub/Sub

频道示例：

```text
chat:events:group:{groupID}
```

收到事件后，本机实例应查询数据库并推送给本机连接。

---

## 十二、IPFS 要求

实现 IPFS 模块抽象：

* 校验 CID
* 注册文件元数据
* 预留 pin 能力接口
* 不要把网关 URL 写死在协议层

消息和文件 API 中都以 `cid` 为核心标识。

---

## 十三、事务与一致性要求

这些操作必须使用事务：

1. 发消息

   * 分配群内 seq
   * 插入消息

2. 编辑消息

   * 锁定消息
   * 写编辑历史
   * 更新消息

3. 删除消息

   * 锁定消息
   * 更新删除状态

4. 更新群消息策略

   * 更新群配置
   * settings_version + 1

群消息 seq 分配方案：

* 在 `groups.last_message_seq` 上做事务内递增
* 使用 `SELECT ... FOR UPDATE` 或等价锁方案
* 不要生成重复 seq

---

## 十四、实现质量要求

代码必须满足：

1. 可编译
2. 可运行
3. 目录清晰
4. 有错误处理
5. 有日志
6. 有配置项
7. 有 README
8. 有 Docker Compose
9. 有示例环境变量文件
10. 关键业务逻辑写清楚注释
11. 不要生成一堆伪代码占位文件
12. 不要只给接口定义不实现
13. 不要过度简化成 toy demo

---

## 十五、输出要求

请直接输出完整项目内容，至少包括：

1. 项目目录树
2. 关键文件完整代码
3. GORM 模型
4. migration
5. 配置加载
6. main.go
7. chi 路由注册
8. middleware
9. auth service
10. group/message/file service
11. websocket hub
12. redis pubsub
13. docker-compose.yml
14. README.md

README 必须包含：

* 启动步骤
* 环境变量说明
* 数据库初始化方式
* WebSocket 使用说明
* API 示例

---

## 十六、额外要求

1. 优先保证架构正确、边界清晰
2. 不要为了省事把所有代码塞到几个文件里
3. 不要擅自改业务规则
4. 不要返回 peer_id 给客户端
5. 不要把 forward 实现成复制快照
6. 不要把 TTL 实现成只对新消息生效
7. 不要跳过 group_message_edits
8. 不要把 Redis 当主数据库
9. 不要把 IPFS 文件二进制存进 PostgreSQL
10. 所有对外 JSON 响应字段命名统一、稳定

---

## 十七、优先实现顺序

请按以下顺序组织代码和交付：

1. 配置、数据库、Redis、基础项目骨架
2. 用户与认证
3. 群与成员
4. 文本消息
5. WebSocket
6. 编辑 / 删除 / 撤回
7. reply / forward
8. file/image/video/voice + IPFS
9. TTL / slow mode
10. README 与 Docker Compose

---

如果实现中某些 libp2p 验签细节需要先抽象接口，也必须把整个认证链路和接入点设计好，不要省略主流程。

---
