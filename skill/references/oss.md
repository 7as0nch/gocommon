# OSS 模块速查

Package：`github.com/7as0nch/gocommon/oss`
框架依赖：**无**（任何 Go 项目可用）
底层：MinIO Go SDK

## Config

```go
type Config struct {
    Endpoint        string  // "play.min.io" 或 "oss.example.com:9000"，不带协议
    AccessKeyID     string
    SecretAccessKey string
    UseSSL          bool
    Region          string  // 可选
    DefaultBucket   string  // 默认桶
}
```

## 创建 + 确保桶存在

```go
import "github.com/7as0nch/gocommon/oss"

cli, err := oss.NewClient(oss.Config{
    Endpoint:        "127.0.0.1:9000",
    AccessKeyID:     "minioadmin",
    SecretAccessKey: "minioadmin",
    DefaultBucket:   "app",
})
if err != nil { return err }

// 第一次启动时确保桶存在
_ = cli.EnsureBucket(ctx, "app", "")
```

## 上传

```go
// 内存字节
info, err := cli.PutObjectBytes(ctx, "", "hello.txt", []byte("hi"), oss.PutObjectOptions{
    ContentType: "text/plain",
})

// 流式（大文件）
info, err := cli.PutObjectStream(ctx, "", "video.mp4", file, fileSize, oss.PutObjectOptions{
    ContentType: "video/mp4",
})
```

bucket 参数传 `""` 时使用 `Config.DefaultBucket`。

## 下载 / 元信息 / 删除

```go
obj, err := cli.GetObject(ctx, "", "hello.txt")
defer obj.Close()
data, _ := io.ReadAll(obj)

stat, err := cli.StatObject(ctx, "", "hello.txt")
fmt.Println(stat.Size, stat.ETag, stat.ContentType)

err = cli.RemoveObject(ctx, "", "hello.txt")
```

## 预签名 URL（直链）

```go
// 给前端下载用
url, _ := cli.PresignedGetURL(ctx, "", "hello.txt", 10*time.Minute, nil)

// 给前端直接上传用（绕过后端）
url, _ := cli.PresignedPutURL(ctx, "", "uploads/avatar.png", 10*time.Minute)
```

## 列举对象

```go
objs, err := cli.ListObjects(ctx, "", oss.ListOptions{
    Prefix:    "uploads/",
    Recursive: true,
    MaxKeys:   100,
})
```

## 未封装的能力

通过 `Raw()` 调用底层 `*minio.Client`：

```go
mc := cli.Raw()
// MakeBucketWithLocation / SetBucketPolicy / CopyObject / Multipart 等高级 API
```

## 常见错误

- `oss: endpoint is required` — Config.Endpoint 为空
- `oss: access key or secret is empty` — 凭证缺失
- MinIO 返回 `NoSuchBucket` — 桶不存在，先 `EnsureBucket`

## 提示

- 不做 Ping 校验：构造时不真正连远端，避免离线开发被阻塞；第一次真实调用 API 时凭证错误自然报错。
- 不内置 CDN 加速：业务侧如有 CDN，把 `PresignedGetURL` 的结果替换 host 即可。
