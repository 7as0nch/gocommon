/* *
 * @Author: chengjiang
 * @Date: 2026-03-17 13:58:54
 * @Description:
**/
package pay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/7as0nch/gocommon/enums"
)

type AlipayConfig struct {
	AppID           string `json:"app_id"`
	AliPayPublicKey string `json:"alipay_public_key"`
	PrivateKey      string `json:"private_key"`
	NotifyURL       string `json:"notify_url"`
	ReturnURL       string `json:"return_url,omitempty"`
	QuitURL         string `json:"quit_url,omitempty"`
	EncryptKey      string `json:"encrypt_key,omitempty"`
	IsProduction    bool   `json:"is_production"`
}

type AppPayOrder struct {
	Subject     string
	OutTradeNo  string
	TotalAmount string
	Body        string
}

type RefundOrder struct {
	OutTradeNo   string
	TradeNo      string
	RefundAmount string
	RefundReason string
	OutRequestNo string
}

// TradeNotification 统一渠道通知结构，service 层只依赖统一语义字段。
type TradeNotification struct {
	NotifyType      string
	OrderNo         string
	ChannelTradeNo  string
	TradeStatus     string
	TotalAmount     string
	ReceiptAmount   string
	BuyerPayAmount  string
	RefundAmount    string
	PaidAt          *time.Time
	RefundedAt      *time.Time
}

// Client 统一支付渠道能力。
// 无配置 client 仅用于从回调里提取我方订单号；有配置 client 负责正式支付、验签与通知解析。
type Client interface {
	ExtractOrderNo(req *http.Request) (string, error)
	AppPay(order AppPayOrder) (string, error)
	WapPay(order AppPayOrder) (string, error)
	PagePay(order AppPayOrder) (string, error)
	Refund(order RefundOrder) (map[string]string, error)
	Query(outTradeNo string) (map[string]string, error)
	ParseTradeNotification(req *http.Request) (*TradeNotification, error)
	AckNotification(w http.ResponseWriter)
}

// NewClient 根据渠道创建基础 client，仅用于回调提取订单号等轻量能力。
func NewClient(channel enums.PayChannel) (Client, error) {
	switch channel {
	case "", enums.PayChannelAlipay:
		return NewAlipayBaseClient(), nil
	default:
		return nil, fmt.Errorf("不支持的支付渠道: %s", channel)
	}
}

// NewClientFromParams 根据渠道解析支付配置并创建正式 client。
func NewClientFromParams(channel enums.PayChannel, rawParams []byte) (Client, error) {
	switch channel {
	case "", enums.PayChannelAlipay:
		var cfg AlipayConfig
		if err := json.Unmarshal(rawParams, &cfg); err != nil {
			return nil, fmt.Errorf("解析支付宝支付配置失败: %w", err)
		}
		return NewAlipayClient(cfg)
	default:
		return nil, fmt.Errorf("不支持的支付渠道: %s", channel)
	}
}