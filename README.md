# ESP RainMaker Self-Hosted Server

兼容 ESP RainMaker Android App 的自托管云平台。

## 快速开始

```bash
# 编译
go build -o rainmaker-server .

# 运行
./rainmaker-server

# 指定配置
./rainmaker-server -config /path/to/config.yaml
```

服务启动后:
- API: `http://localhost:8080/v1/`
- 管理后台: `http://localhost:8080/admin/`
- 默认管理员: admin / admin123

## App 对接

在 Android 项目的 `local.properties` 中添加:

```properties
baseUrl=http://你的服务器IP:8080
claimBaseUrl=http://你的服务器IP:8080
tokenUrl=http://你的服务器IP:8080/v1/token
```

## 配置

编辑 `config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

jwt:
  secret: "your-secret-key"
  access_token_ttl: 3600
  refresh_token_ttl: 2592000
```

## API 端点

所有端点前缀: `/v1`

| 类别 | 端点 | 说明 |
|------|------|------|
| 认证 | POST /login | 登录/刷新Token |
| 认证 | POST /user | 注册用户 |
| 认证 | POST /logout | 登出 |
| 节点 | GET /user/nodes | 获取节点列表 |
| 节点 | PUT /user/nodes | 添加/删除节点 |
| 节点 | GET /user/nodes/params | 获取参数 |
| 节点 | PUT /user/nodes/params | 更新参数 |
| 认领 | POST /claim/initiate | 设备认领 |
| 映射 | POST /user/nodes/mapping/initiate | 设备映射 |
| 共享 | PUT /user/nodes/sharing | 共享设备 |
| 分组 | POST /user/node_group | 创建分组 |
| 自动化 | POST /user/node_automation | 创建自动化 |
| OTA | POST /user/nodes/ota_update | OTA升级 |
