// Package enums 定义 gocommon 各子模块共享的渠道/类型常量。
// 约定：对外暴露的常量统一用 string 别名类型，避免直接裸字符串，便于 switch 穷举与检索。
package enums

// PayChannel 支付渠道枚举，与 utils/pay 配合使用。
type PayChannel string

const (
	// PayChannelAlipay 支付宝。
	PayChannelAlipay PayChannel = "alipay"
	// PayChannelWechat 微信支付（预留，后续接入时补充实现）。
	PayChannelWechat PayChannel = "wechat"
)

// SMSChannel 短信渠道枚举，与 utils/sms 配合使用。
type SMSChannel string

const (
	// SMSChannelAli 阿里云短信。
	SMSChannelAli SMSChannel = "aliyun"
	// SMSChannelTencent 腾讯云短信（预留）。
	SMSChannelTencent SMSChannel = "tencent"
)

// IdentityType 登录/身份类型枚举，用于 auth/identityprovider。
// 这里给出一组常见的默认值，业务项目可按需在自己的 model 层再扩展。
type IdentityType string

const (
	IdentityTypeAccount IdentityType = "account"
	IdentityTypePhone   IdentityType = "phone"
	IdentityTypeEmail   IdentityType = "email"
	IdentityTypeQQ      IdentityType = "qq"
	IdentityTypeWechat  IdentityType = "wechat"
)
