# 短信模块速查

Package：`github.com/7as0nch/gocommon/utils/sms`
框架依赖：**无**（任何 Go 项目可用）
当前已实现：**阿里云**
路线图：腾讯云

## Config

```go
type AliyunConfig struct {
    AccessKeyID     string  `json:"access_key_id"`
    AccessKeySecret string  `json:"access_key_secret"`
    Endpoint        string  `json:"endpoint"`     // 默认 "dysmsapi.aliyuncs.com"
    RegionID        string  `json:"region_id"`    // 默认 "cn-hangzhou"
    SignName        string  `json:"sign_name"`    // 短信签名
    TemplateCode    string  `json:"template_code"`// 模板 ID
}
```

## 创建 + 发送

```go
import (
    "github.com/7as0nch/gocommon/utils/sms"
    "github.com/7as0nch/gocommon/enums"
)

cli, err := sms.NewClientWithConfig(enums.SMSChannelAli, sms.AliyunConfig{
    AccessKeyID:     "...",
    AccessKeySecret: "...",
    SignName:        "你的签名",
    TemplateCode:    "SMS_xxxx",
})

err = cli.Send(ctx, &sms.Request{
    PhoneNumber:    "13800000000",
    TemplateParams: `{"code":"123456","minute":"5"}`,  // 模板变量 JSON
    // SignName/TemplateCode 留空时取 Config 默认值
})
```

## 验证码场景配合 Redis

```go
import "github.com/7as0nch/gocommon/consts"

code := utils.CreateCaptcha(6)
_ = rdb.Set(ctx, fmt.Sprintf(consts.USER_AUTH_SMS_CAPTCHA_KEY, phone), code, 5*time.Minute)

_ = sms.Send(ctx, &sms.Request{
    PhoneNumber:    phone,
    TemplateParams: fmt.Sprintf(`{"code":"%s"}`, code),
})
```

校验：

```go
saved, _ := rdb.Get(ctx, fmt.Sprintf(consts.USER_AUTH_SMS_CAPTCHA_KEY, phone))
if saved != userInput { return ErrInvalidCaptcha }
_, _ = rdb.Del(ctx, fmt.Sprintf(consts.USER_AUTH_SMS_CAPTCHA_KEY, phone))
```

## 防刷限流

建议配合 Redis 限流：每分钟同一手机号 ≤ 1 条；每日 ≤ 5 条。
路线图 P1 的 `redis/ratelimit.go` 会提供令牌桶/滑窗实现，届时可直接复用。

## 常见错误

- `isv.MOBILE_NUMBER_ILLEGAL` — 手机号格式错误
- `isv.OUT_OF_SERVICE` — 阿里云账户欠费
- `isv.SMS_TEMPLATE_ILLEGAL` — 模板未审核或不属于该 AccessKey
- `isv.SMS_SIGNATURE_ILLEGAL` — 签名未审核或不匹配
