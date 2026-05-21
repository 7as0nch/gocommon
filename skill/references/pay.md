# 支付模块速查

Package：`github.com/7as0nch/gocommon/utils/pay`
框架依赖：**无**（任何 Go 项目可用）
当前已实现：**支付宝**（基于 `github.com/smartwalle/alipay/v3`）
路线图：微信支付

## Config

```go
type AlipayConfig struct {
    AppID           string  `json:"app_id"`
    AliPayPublicKey string  `json:"alipay_public_key"`
    PrivateKey      string  `json:"private_key"`
    NotifyURL       string  `json:"notify_url"`
    ReturnURL       string  `json:"return_url,omitempty"`
    QuitURL         string  `json:"quit_url,omitempty"`
    EncryptKey      string  `json:"encrypt_key,omitempty"`
    IsProduction    bool    `json:"is_production"`
}
```

## 创建客户端

```go
import (
    "github.com/7as0nch/gocommon/utils/pay"
    "github.com/7as0nch/gocommon/enums"
)

// 方式 1：直接 New
cli, err := pay.NewAlipayClient(pay.AlipayConfig{
    AppID: "...", PrivateKey: "...", AliPayPublicKey: "...",
    NotifyURL: "https://your.host/api/pay/alipay/notify",
    IsProduction: true,
})

// 方式 2：基于渠道枚举（多渠道场景）
cfg, _ := json.Marshal(alipayCfg)
cli, _ := pay.NewClientFromParams(enums.PayChannelAlipay, cfg)

// 方式 3：仅做回调订单号提取（不需要完整 SDK）
base, _ := pay.NewClient(enums.PayChannelAlipay)
orderNo, _ := base.ExtractOrderNo(req)
```

## 下单（APP / H5 / Web）

```go
order := pay.AppPayOrder{
    Subject:     "VIP 月卡",
    OutTradeNo:  "ORD20260101001",
    TotalAmount: "9.90",
    Body:        "VIP 月卡 30 天",
}

orderStr, err := cli.AppPay(order)      // 返回 App SDK 用的订单串
url, err     := cli.WapPay(order)       // 返回 H5 跳转 URL
url, err     := cli.PagePay(order)      // 返回 PC Web 表单 URL
```

## 退款 / 查询

```go
refundResp, err := cli.Refund(pay.RefundOrder{
    OutTradeNo:   "ORD20260101001",
    RefundAmount: "9.90",
    RefundReason: "用户取消",
    OutRequestNo: "REF20260101001",  // 一次退款的唯一标识
})

queryResp, err := cli.Query("ORD20260101001")
```

## 异步通知

支付宝回调到 `NotifyURL`，业务侧：

```go
func (s *PayService) AlipayNotify(w http.ResponseWriter, req *http.Request) {
    n, err := cli.ParseTradeNotification(req)  // 内部已验签
    if err != nil { http.Error(w, "verify failed", 400); return }

    // n.OrderNo, n.ChannelTradeNo, n.TradeStatus, n.TotalAmount, n.PaidAt ...
    if n.TradeStatus == "TRADE_SUCCESS" {
        // 更新订单为已支付
    }
    cli.AckNotification(w)  // 写回 "success"
}
```

`TradeNotification` 是渠道无关的统一结构：

```go
type TradeNotification struct {
    NotifyType, OrderNo, ChannelTradeNo, TradeStatus string
    TotalAmount, ReceiptAmount, BuyerPayAmount, RefundAmount string
    PaidAt, RefundedAt *time.Time
}
```

## 防重复回调（Redis 锁）

推荐配合 `consts/redis.go:USER_PAY_LOCK_KEY` 使用 SetNX 做幂等：

```go
ok, _ := rdb.SetNX(ctx, fmt.Sprintf(consts.USER_PAY_LOCK_KEY, n.OrderNo, "notify"), "1", 60*time.Second)
if !ok { return }  // 重复回调
```

## 常见错误

- 验签失败 — 检查 `AliPayPublicKey` 是否是从支付宝后台复制的"支付宝公钥"而非自己的应用公钥
- 沙箱与生产不一致 — 切换 `IsProduction` 时必须同步更换公钥
