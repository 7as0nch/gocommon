package consts

/* *
 * @Author: chengjiang
 * @Date: 2026-04-20 11:58:06
 * @Description:
**/

import "time"

const (
	PAY_CONFIG_CACHE_KEY  = "pay_config"             // 支付配置缓存 key
	API_LIMIT_KEY         = "api_limit"              // API 限流缓存 key
	APPLICATION_CACHE_KEY = "application:%s"         // 应用缓存 key %s:app_key
	APPICATION_DICT_KEY   = "application_dict:%d:%s" // 应用字典缓存 key %d:appid %s:dict_code
)

const (
	// 用户认证相关
	USER_AUTH_LOCK_KEY         = "user:auth:lock:%s:%s" // 用户认证(注册/登录)锁 key: ip:identifier
	USER_AUTH_LOCK_EXPIRE      = 5 * time.Second
	USER_AUTH_USER_TOKENS_KEY  = "user:auth:tokens:%s" // 用户 token 哈希 key: userID | guestDeviceTicket
	USER_AUTH_USER_INFO_KEY    = "user:auth:info:%d"   // 用户信息缓存 key: userID
	USER_AUTH_USER_INFO_TTL    = 5 * time.Minute
	USER_AUTH_LOGIN_TRY_KEY    = "user:auth:login:try:%s:%s" // 用户登录尝试次数 key: ip:identifier
	USER_AUTH_LOGIN_TRY_EXPIRE = 5 * time.Minute
	// 短信验证码相关
	USER_AUTH_SMS_CAPTCHA_KEY           = "user:auth:sms:captcha:%s"              // 短信验证码 key:phone
	USER_AUTH_SMS_LIMIT_IP_KEY          = "user:auth:sms:limit:ip:%d:%s"          // 短信发送限制 key: appid:ip
	USER_AUTH_SMS_LIMIT_USER_DEVICE_KEY = "user:auth:sms:limit:user-device:%d:%s" // 短信发送限制 key: appid:deviceValue
	USER_AUTH_SLIDE_STATE_KEY           = "user:auth:slide:%s"                    // 滑块状态 key: token/credential
	USER_AUTH_SLIDE_CAPTCHA_TTL         = 5 * time.Minute
	USER_AUTH_SMS_LIMIT_IP_TTL          = 24 * time.Hour
	USER_AUTH_SMS_LIMIT_USER_DEVICE_TTL = 24 * time.Hour
	USER_AUTH_OAUTH_STATE_KEY           = "user:auth:oauth:state:%s"
	USER_AUTH_OAUTH_TICKET_KEY          = "user:auth:oauth:ticket:%s"
	USER_AUTH_OAUTH_STATE_TTL           = 10 * time.Minute
	USER_AUTH_OAUTH_TICKET_TTL          = 1 * time.Minute
	// 支付相关
	USER_PAY_LOCK_KEY            = "user:pay:lock:%s:%s" // 支付防抖锁 key: userID, appid+payChannel+productIDs
	USER_PAY_LOCK_TTL            = 5 * time.Second       // 支付防抖窗口
	USER_PAY_ORDER_KEY           = "user:pay:order:%s"   // 支付订单缓存 key: orderNo
	USER_PAY_ORDER_TTL           = 15 * time.Minute      // 支付订单缓存有效期
	USER_PAY_ORDER_TIMEOUT       = 15 * time.Minute      // 订单超时时间
	USER_PAY_TIMEOUT_RETRY_DELAY = 5 * time.Second       // 处理失败后的重试延迟
)
