# MeshChat Server API 文档

这份文档面向程序接入方，描述 MeshChat Server 当前可用的 HTTP API、WebSocket 协议、认证方式和主要数据结构。

## 1. 基础约定

### 1.1 服务地址

- HTTP Base URL: `http://<host>:8080`
- Admin Base URL: `http://<host>:8081`
- WebSocket URL: `ws://<host>:8080/ws?token=<jwt>`
- IPFS Gateway: `http://<host>:8081/ipfs/<cid>`

### 1.2 数据格式

- 请求体和响应体均为 `application/json`
- 时间字段统一使用 RFC3339 格式的 UTC 时间字符串
- 除登录相关接口外，所有 HTTP API 都要求 Bearer Token
- 用户资料相关接口会返回 `peer_id`，用于按稳定标识更新资料
- 群创建权限受 `SERVER_MODE` 控制

### 1.3 鉴权头

```http
Authorization: Bearer <jwt>
```

### 1.4 错误响应格式

所有非 2xx 响应统一格式：

```json
{
  "error": {
    "code": "forbidden",
    "message": "missing delete permission"
  }
}
```

常见错误码：

- `missing_token`
- `invalid_token`
- `invalid_json`
- `invalid_payload`
- `forbidden`
- `server_admin_required`
- `group_not_found`
- `group_closed`
- `member_not_found`
- `message_not_found`
- `reference_not_visible`
- `slow_mode`
- `retract_window_expired`

### 1.5 服务器模式

通过环境变量 `SERVER_MODE` 控制服务器建群策略：

- `public`: 所有已登录用户都可以创建群
- `restricted`: 只有服务器管理员可以创建群

服务器管理员通过 `SERVER_ADMIN_PEER_IDS` 中配置的 `peer_id` 判定。

### 1.6 服务器信息接口

`GET /server/info`

无需鉴权，返回：

```json
{
  "server_mode": "public"
}
```

## 2. 认证流程

### 2.1 获取 challenge

`POST /auth/challenge`

请求体：

```json
{
  "peer_id": "12D3KooW..."
}
```

响应体：

```json
{
  "challenge_id": "3e0d77dc-bf44-4532-b7f5-1f10e3ad5671",
  "challenge": "meshchat login\nchallenge_id=...\npeer_id=...\nexpires_at=...",
  "expires_at": "2026-03-28T12:00:00Z"
}
```

### 2.2 提交签名登录

`POST /auth/login`

请求体：

```json
{
  "peer_id": "12D3KooW...",
  "challenge_id": "3e0d77dc-bf44-4532-b7f5-1f10e3ad5671",
  "signature": "BASE64_SIGNATURE",
  "public_key": "BASE64_MARSHALLED_LIBP2P_PUBLIC_KEY"
}
```

响应体：

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "username": "user_xxx",
    "display_name": "user_xxx",
    "avatar_cid": "",
    "bio": "",
    "profile_version": 1,
    "status": "active",
    "created_at": "2026-03-28T12:00:00Z",
    "updated_at": "2026-03-28T12:00:00Z"
  }
}
```

## 3. 管理后台

管理后台运行在独立端口，默认 `ADMIN_HTTP_ADDR=:8081`。它使用 `ADMIN_USERNAME` / `ADMIN_PASSWORD` 登录，返回独立的 admin JWT，不和前台用户 token 混用。

浏览器直接访问后台端口根路径 `/` 即可打开管理页面。

### 3.1 登录

`POST /admin/login`

请求体：

```json
{
  "username": "admin",
  "password": "admin123456"
}
```

响应体：

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "username": "admin"
}
```

### 3.2 常用接口

所有以下接口都需要 `Authorization: Bearer <admin-jwt>`。

- `GET /admin/me`
- `GET /admin/users`
- `PATCH /admin/users/{peer_id}`
- `GET /admin/groups`
- `POST /admin/groups`
- `GET /admin/groups/{group_id}`
- `PATCH /admin/groups/{group_id}`
- `POST /admin/groups/{group_id}/members/{user_id}/admin`
- `POST /admin/groups/{group_id}/transfer-owner`
- `POST /admin/groups/{group_id}/dissolve`
- `GET /admin/groups/{group_id}/members`
- `GET /admin/groups/{group_id}/messages`

说明：

- `GET /admin/users` 返回用户列表，包含 `peer_id`
- `PATCH /admin/users/{peer_id}` 按 `peer_id` 修改用户资料
- `GET /admin/groups` 返回群列表
- `POST /admin/groups` 创建群聊
- `PATCH /admin/groups/{group_id}` 修改群资料和群配置
- `POST /admin/groups/{group_id}/members/{user_id}/admin` 设置或取消某个成员的群管理员权限
- `POST /admin/groups/{group_id}/transfer-owner` 将群主转让给本群其他活跃成员
- `POST /admin/groups/{group_id}/dissolve` 解散群聊
- `GET /admin/groups/{group_id}/members` 查看群成员
- `GET /admin/groups/{group_id}/messages` 查看群聊天记录

### 3.3 管理员操作请求体

#### 修改用户资料

`PATCH /admin/users/{peer_id}`

请求体：

```json
{
  "display_name": "Alice",
  "avatar_cid": "bafy...",
  "bio": "hello",
  "status": "active"
}
```

`peer_id` 是稳定标识，用来定位要更新的用户。

#### 设置群管理员

`POST /admin/groups/{group_id}/members/{user_id}/admin`

请求体：

```json
{
  "is_admin": true
}
```

将成员设为群管理员。

```json
{
  "is_admin": false
}
```

取消群管理员权限，目标成员会回到普通成员角色。

#### 转让群主

`POST /admin/groups/{group_id}/transfer-owner`

请求体：

```json
{
  "user_id": 123
}
```

将群主转让给指定的本群活跃成员。转让后原群主自动降为管理员。

## 4. 公共数据结构

### 3.1 User

```json
{
  "id": 1,
  "username": "alice",
  "display_name": "Alice",
  "avatar_cid": "bafy...",
  "bio": "hello",
  "profile_version": 2,
  "status": "active",
  "created_at": "2026-03-28T12:00:00Z",
  "updated_at": "2026-03-28T12:10:00Z"
}
```

### 3.2 Group

```json
{
  "group_id": "7a5682a3-8ff9-42a9-85b9-1d4c41d95c20",
  "title": "MeshChat",
  "about": "backend group",
  "avatar_cid": "bafy...",
  "owner_user": {},
  "member_list_visibility": "hidden",
  "join_mode": "invite_only",
  "default_permissions": 2047,
  "message_ttl_seconds": 0,
  "message_retract_seconds": 300,
  "message_cooldown_seconds": 0,
  "last_message_seq": 12,
  "last_message_timestamp": 1711632000,
  "settings_version": 3,
  "status": "active",
  "effective_permissions": 9223372036854775807,
  "created_at": "2026-03-28T12:00:00Z",
  "updated_at": "2026-03-28T12:10:00Z"
}
```

### 3.3 GroupMember

```json
{
  "user": {},
  "role": "member",
  "status": "active",
  "title": "QA",
  "joined_at": "2026-03-28T12:00:00Z",
  "muted_until": null,
  "permissions_allow": 0,
  "permissions_deny": 0,
  "effective_permissions": 2047
}
```

### 3.4 Message

```json
{
  "group_id": "7a5682a3-8ff9-42a9-85b9-1d4c41d95c20",
  "message_id": "5ecf8018-027d-4e72-bd34-c66773178957",
  "seq": 12,
  "content_type": "text",
  "payload": {
    "text": "hello"
  },
  "reply_to_message_id": null,
  "forward_from_message_id": null,
  "forward": null,
  "sender": {},
  "status": "normal",
  "edit_count": 0,
  "last_edited_at": null,
  "delete_reason": null,
  "deleted_at": null,
  "created_at": "2026-03-28T12:00:00Z"
}
```

### 3.5 Forward 引用对象

当消息 `content_type = forward` 时，响应中的 `forward` 字段描述原消息状态。

可用状态：

- `ok`
- `deleted_or_expired`
- `unavailable`
- `missing`

示例：

```json
{
  "state": "ok",
  "message": {
    "group_id": "7a5682a3-8ff9-42a9-85b9-1d4c41d95c20",
    "message_id": "5ecf8018-027d-4e72-bd34-c66773178957",
    "seq": 9,
    "content_type": "text",
    "payload": {
      "text": "origin"
    },
    "sender": {},
    "status": "normal",
    "created_at": "2026-03-28T12:00:00Z"
  }
}
```

### 3.6 枚举值

#### role

- `owner`
- `admin`
- `member`
- `restricted`

#### member status

- `active`
- `left`
- `kicked`
- `banned`

#### group status

- `active`
- `closed`

#### member_list_visibility

- `visible`
- `hidden`

#### join_mode

- `invite_only`
- `open`

#### message content_type

- `text`
- `image`
- `video`
- `voice`
- `file`
- `forward`

#### message status

- `normal`
- `deleted`

#### delete_reason

- `self_retracted`
- `admin_removed`

## 4. 权限位

权限使用 `int64` 位掩码：

- `PERM_SEND_TEXT = 1`
- `PERM_SEND_IMAGE = 2`
- `PERM_SEND_VIDEO = 4`
- `PERM_SEND_VOICE = 8`
- `PERM_SEND_FILE = 16`
- `PERM_REPLY = 32`
- `PERM_FORWARD = 64`
- `PERM_EDIT_OWN_MESSAGES = 128`
- `PERM_DELETE_OWN_MESSAGES = 256`
- `PERM_DELETE_MESSAGES = 512`
- `PERM_VIEW_MEMBERS = 1024`
- `PERM_BYPASS_SLOWMODE = 2048`
- `PERM_MANAGE_MESSAGE_POLICY = 4096`
- `PERM_EDIT_GROUP_INFO = 8192`
- `PERM_MUTE_MEMBERS = 16384`
- `PERM_BAN_MEMBERS = 32768`
- `PERM_SET_MEMBER_PERMISSIONS = 65536`

服务端计算逻辑：

```text
effective_permissions =
(role_base_permissions OR permissions_allow) AND (~permissions_deny)
```

## 5. 消息 payload 结构

### 5.1 text

```json
{
  "text": "hello"
}
```

### 5.2 image

```json
{
  "cid": "bafy...",
  "mime_type": "image/jpeg",
  "size": 123456,
  "width": 1280,
  "height": 720,
  "caption": "cover",
  "thumbnail_cid": "bafy..."
}
```

### 5.3 video

```json
{
  "cid": "bafy...",
  "mime_type": "video/mp4",
  "size": 12345678,
  "width": 1920,
  "height": 1080,
  "duration": 35,
  "caption": "clip",
  "thumbnail_cid": "bafy..."
}
```

### 5.4 voice

```json
{
  "cid": "bafy...",
  "mime_type": "audio/ogg",
  "size": 45678,
  "duration": 12,
  "waveform": "base64..."
}
```

### 5.5 file

```json
{
  "cid": "bafy...",
  "mime_type": "application/pdf",
  "size": 456789,
  "file_name": "spec.pdf",
  "caption": "latest spec"
}
```

### 5.6 forward

```json
{
  "comment": "optional text"
}
```

## 6. HTTP API

## 6.1 健康检查

`GET /healthz`

响应体：

```json
{
  "status": "ok"
}
```

## 6.2 用户资料

### GET /me/profile

返回当前登录用户资料。

返回的用户对象包含 `peer_id`。

### PATCH /users/{peer_id}/profile

按 `peer_id` 更新当前登录用户资料。

`peer_id` 必须和当前登录用户一致。

请求体：

```json
{
  "display_name": "Alice",
  "avatar_cid": "bafy...",
  "bio": "hello",
  "status": "active"
}
```

### GET /me/groups

返回当前用户所有 `active` 加入的群聊。

可选查询参数：

- `limit`
- `offset`

返回 `GroupView` 列表。

## 6.3 群组

### POST /groups

权限要求：

- `SERVER_MODE=public` 时，任意已登录用户可调用
- `SERVER_MODE=restricted` 时，调用者必须是服务器管理员

请求体：

```json
{
  "title": "MeshChat",
  "about": "backend group",
  "avatar_cid": "bafy...",
  "member_list_visibility": "hidden",
  "join_mode": "invite_only",
  "default_permissions": 2047
}
```

返回 `Group` 对象。

### GET /groups/{group_id}

返回群详情。

### POST /groups/{group_id}/join

权限要求：

- 群必须处于 `active`
- 群的 `join_mode` 必须是 `open`

无请求体。

说明：

- 用户加入后才有该群的发言权限
- 重复加入是幂等的
- `banned` 成员不能通过该接口重新加入

返回当前成员对象。

### POST /groups/{group_id}/members/{user_id}/invite

权限要求：

- 调用者必须是服务器管理员，或者该群的活跃群主/管理员
- 目标用户必须存在

无请求体。

说明：

- 该接口用于主动邀请用户进入群聊
- 当前实现不会生成“待接受邀请”，邀请成功后会直接将目标用户设为 `active`
- 如果目标用户之前是 `left` 或 `kicked`，会被重新激活
- `banned` 成员不能通过该接口重新加入

返回当前成员对象。

### POST /groups/{group_id}/members/invite

权限要求：

- 调用者必须是服务器管理员，或者该群的活跃群主/管理员
- `peer_ids` 中的目标用户如果不存在，服务端会自动创建本地用户记录

请求体：

```json
{
  "peer_ids": [
    "12D3KooW...",
    "12D3KooW..."
  ]
}
```

说明：

- 该接口支持一次邀请多个好友
- 每个 `peer_id` 都会在服务器内确保存在对应用户
- 当前实现不会生成“待接受邀请”，邀请成功后会直接将目标用户设为 `active`
- 如果目标用户之前是 `left` 或 `kicked`，会被重新激活
- `banned` 成员不能通过该接口重新加入

返回当前被处理成员列表，顺序与 `peer_ids` 去重后的顺序一致。

### POST /groups/{group_id}/leave

权限要求：

- 调用者必须是该群的活跃成员

无请求体。

说明：

- 退出后成员状态会变为 `left`
- 群主不能直接退出，必须先转让群主
- 公开群在退出后仍可重新 `join`

返回当前成员对象。

### PATCH /groups/{group_id}

请求体：

```json
{
  "title": "MeshChat v2",
  "about": "new about",
  "avatar_cid": "bafy...",
  "member_list_visibility": "visible",
  "join_mode": "open",
  "status": "active"
}
```

返回更新后的 `Group` 对象。

### POST /groups/{group_id}/transfer-owner

权限要求：

- 调用者必须是当前群主

请求体：

```json
{
  "user_id": 12
}
```

说明：

- 目标用户必须是该群的活跃成员
- 转让后新群主角色变为 `owner`
- 原群主角色会自动降为 `admin`

返回更新后的 `Group` 对象。

### POST /groups/{group_id}/dissolve

权限要求：

- 调用者必须是服务器管理员

无请求体。

返回：

```json
{
  "group_id": "7a5682a3-8ff9-42a9-85b9-1d4c41d95c20",
  "status": "closed",
  "settings_version": 4,
  "updated_at": "2026-03-28T12:20:00Z"
}
```

说明：

- 群解散后 `status = closed`
- 已关闭群不再允许普通成员继续进行群内访问和消息操作

### PATCH /groups/{group_id}/message-policy

请求体：

```json
{
  "message_ttl_seconds": 86400,
  "message_retract_seconds": 300,
  "message_cooldown_seconds": 10
}
```

返回更新后的 `Group` 对象。

说明：

- `message_ttl_seconds = 0` 表示消息永久可见
- `message_retract_seconds = 0` 表示不限制撤回窗口
- `message_cooldown_seconds = 0` 表示关闭 slow mode

## 6.4 成员

### GET /groups/{group_id}/members

返回成员数组：

```json
[
  {
    "user": {},
    "role": "member",
    "status": "active",
    "title": "",
    "joined_at": "2026-03-28T12:00:00Z",
    "muted_until": null,
    "permissions_allow": 0,
    "permissions_deny": 0,
    "effective_permissions": 2047
  }
]
```

如果群隐藏成员列表，则需要调用者有 `PERM_VIEW_MEMBERS`。

### PATCH /groups/{group_id}/members/{user_id}/permissions

这里的 `user_id` 是服务端用户 ID，不是 `peer_id`。

请求体：

```json
{
  "permissions_allow": 32,
  "permissions_deny": 16
}
```

返回更新后的 `GroupMember` 对象。

### POST /groups/{group_id}/members/{user_id}/admin

权限要求：

- 调用者必须是服务器管理员，或该群的群主

请求体：

```json
{
  "is_admin": true
}
```

说明：

- `is_admin = true` 表示将该成员设为 `admin`
- `is_admin = false` 表示取消该成员的管理员身份
- `owner` 不能通过该接口修改
- 群主可以为自己的群设置新管理员

返回更新后的 `GroupMember` 对象。

### POST /groups/{group_id}/members/{user_id}/mute

请求体：

```json
{
  "duration_seconds": 600
}
```

说明：

- `duration_seconds > 0` 表示禁言到未来某个时间点
- `duration_seconds = 0` 表示解除禁言

返回更新后的 `GroupMember` 对象。

### POST /groups/{group_id}/members/{user_id}/ban

无请求体。

返回更新后的 `GroupMember` 对象，`status` 会变为 `banned`。

## 6.5 消息

### GET /groups/{group_id}/messages

查询参数：

- `before_seq`: 可选，按群内消息序号向前翻页
- `limit`: 可选，默认 50，最大 100

示例：

```text
GET /groups/{group_id}/messages?before_seq=100&limit=20
```

返回消息数组，服务端会自动过滤当前 TTL 下已经过期的消息。

### POST /groups/{group_id}/messages

请求体：

```json
{
  "content_type": "text",
  "payload": {
    "text": "hello"
  },
  "reply_to_message_id": null,
  "forward_from_message_id": null,
  "signature": ""
}
```

规则：

- `reply_to_message_id` 和 `forward_from_message_id` 互斥
- `forward` 消息时应设置 `content_type = forward`
- `reply` 只能引用同群消息
- `forward` 可跨群，但原消息必须对当前用户可见
- 文件类消息只存 CID 和元数据，不上传二进制

返回新建后的 `Message` 对象。

### PATCH /groups/{group_id}/messages/{message_id}

请求体：

```json
{
  "payload": {
    "text": "edited text"
  }
}
```

规则：

- 仅发送者本人可以编辑
- 只允许编辑 payload
- 不允许修改 sender、content_type、reply_to、forward_from
- 已删除消息不可编辑

返回编辑后的 `Message` 对象。

### POST /groups/{group_id}/messages/{message_id}/retract

无请求体。

规则：

- 仅允许撤回自己的消息
- 受 `message_retract_seconds` 约束

返回删除后的 `Message` 对象，`status = deleted`，`delete_reason = self_retracted`。

### POST /groups/{group_id}/messages/{message_id}/delete

无请求体。

规则：

- 需要 `PERM_DELETE_MESSAGES`
- 不可删除 owner 的消息
- 低权限管理员不可删除更高权限成员的消息

返回删除后的 `Message` 对象，`status = deleted`，`delete_reason = admin_removed`。

## 6.6 文件元数据

### POST /files

请求体：

```json
{
  "cid": "bafy...",
  "mime_type": "application/pdf",
  "size": 456789,
  "width": null,
  "height": null,
  "duration_seconds": null,
  "file_name": "spec.pdf",
  "thumbnail_cid": ""
}
```

返回：

```json
{
  "id": 1,
  "cid": "bafy...",
  "mime_type": "application/pdf",
  "size": 456789,
  "width": null,
  "height": null,
  "duration_seconds": null,
  "file_name": "spec.pdf",
  "thumbnail_cid": "",
  "created_at": "2026-03-28T12:00:00Z"
}
```

说明：

- 该接口只登记元数据，不上传文件内容
- 文件内容由客户端自行写入 IPFS
- 客户端可通过 CID 走 IPFS 网络或网关拉取内容

## 7. WebSocket 协议

## 7.1 建立连接

```text
GET /ws?token=<jwt>
```

也可以通过 `Authorization: Bearer <jwt>` 传 token。

## 7.2 客户端命令

### 订阅群

```json
{
  "action": "subscribe",
  "group_ids": [
    "7a5682a3-8ff9-42a9-85b9-1d4c41d95c20",
    "c0d6496f-20e4-4c84-9f8a-8f60ea3613e2"
  ]
}
```

### 取消订阅

```json
{
  "action": "unsubscribe",
  "group_ids": [
    "7a5682a3-8ff9-42a9-85b9-1d4c41d95c20"
  ]
}
```

### 订阅确认响应

```json
{
  "type": "subscription.updated",
  "data": {
    "group_ids": [
      "7a5682a3-8ff9-42a9-85b9-1d4c41d95c20"
    ]
  }
}
```

## 7.3 服务端事件格式

统一格式：

```json
{
  "type": "group.message.created",
  "data": {}
}
```

支持事件：

- `group.message.created`
- `group.message.edited`
- `group.message.deleted`
- `group.settings.updated`
- `group.member.updated`

各事件的 `data` 分别对应：

- 消息事件：`Message`
- 群设置事件：`Group`
- 成员事件：`GroupMember`

## 8. 行为约束

接入方需要特别注意以下规则：

- `peer_id` 只用于 challenge 登录，不会出现在群成员和消息数据里
- 服务器管理员由环境变量 `SERVER_ADMIN_PEER_IDS` 中的 `peer_id` 决定
- `SERVER_MODE=public` 时普通用户也可创建群，`restricted` 时仅服务器管理员可创建群
- `message_id`、`group_id` 使用 UUID 字符串
- `user_id` 使用服务端内部整型 ID
- `forward` 是引用型转发，不复制原消息内容
- 原消息被编辑后，`forward` 的 `forward.message.payload` 会反映最新内容
- 原消息删除或过期后，`forward.state` 会变为不可用状态
- 消息 TTL 是动态策略，旧消息也会受新 TTL 影响
- slow mode 作用于 `(group_id, user_id)` 维度
- 文件消息和文件注册接口只保存 CID 与元数据

## 9. 推荐对接顺序

建议其它程序按下面顺序接入：

1. 调 `POST /auth/challenge`
2. 用 libp2p 私钥签名 challenge
3. 调 `POST /auth/login` 获取 JWT
4. 拉取 `/me/profile`
5. 拉取群详情和消息列表
6. 建立 `/ws` 长连接并订阅需要的群
7. 发消息、编辑消息、撤回消息、处理实时事件
