// Package auth
// @author chengjiang
// @note  JWT 相关定义：claims、options、错误、context 读写。
//        已升级到 github.com/golang-jwt/jwt/v5。
package auth

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/golang-jwt/jwt/v5"
)

type authKey struct{}

const (
	// BearerWord the bearer key word for authorization
	BearerWord string = "Bearer"

	// BearerFormat authorization token format
	BearerFormat string = "Bearer %s"

	// AuthorizationKey holds the key used to store the JWT Token in the request tokenHeader.
	AuthorizationKey string = "Authorization"

	// Reason holds the error Reason.
	Reason string = "UNAUTHORIZED"
)

// 错误定义，统一使用 kratos 的 errors.Unauthorized 方便上层透传 HTTP 401。
var (
	ErrMissingJwtToken        = errors.Unauthorized(Reason, "JWT token is missing")
	ErrMissingKeyFunc         = errors.Unauthorized(Reason, "keyFunc is missing")
	ErrTokenInvalid           = errors.Unauthorized(Reason, "Token is invalid")
	ErrTokenExpired           = errors.Unauthorized(Reason, "JWT token has expired")
	ErrTokenParseFail         = errors.Unauthorized(Reason, "Fail to parse JWT token")
	ErrUnSupportSigningMethod = errors.Unauthorized(Reason, "Wrong signing method")
	ErrWrongContext           = errors.Unauthorized(Reason, "Wrong context for middleware")
	ErrNeedTokenProvider      = errors.Unauthorized(Reason, "Token provider is missing")
	ErrSignToken              = errors.Unauthorized(Reason, "Can not sign token. Is the key correct?")
	ErrGetKey                 = errors.Unauthorized(Reason, "Can not get key while signing token")
)

// Option 中间件配置项。
type Option func(*options)

// options 内部配置结构。
type options struct {
	signingMethod jwt.SigningMethod
	claims        func() jwt.Claims
	tokenHeader   map[string]interface{}
}

// JwtClaims 默认的业务 claims，包含用户基本信息。
// 业务项目可自定义 claims 并通过 WithClaims 注入，不强制使用此结构。
type JwtClaims struct {
	UserId    int64  `json:"UserId"`
	UserName  string `json:"UserName"`
	UserPhone string `json:"UserPhone"`
	jwt.RegisteredClaims
}

// WithSigningMethod with signing method option.
func WithSigningMethod(method jwt.SigningMethod) Option {
	return func(o *options) {
		o.signingMethod = method
	}
}

// WithClaims with customer claim.
// 服务端使用时 f 需要每次返回新 claims 实例以规避并发写问题；
// 客户端使用时 f 可以返回单例以提高性能。
func WithClaims(f func() jwt.Claims) Option {
	return func(o *options) {
		o.claims = f
	}
}

// WithTokenHeader 客户端自定义 token header。
func WithTokenHeader(header map[string]interface{}) Option {
	return func(o *options) {
		o.tokenHeader = header
	}
}

// NewContext put auth info into context.
func NewContext(ctx context.Context, info jwt.Claims) context.Context {
	return context.WithValue(ctx, authKey{}, info)
}

// FromContext extract auth info from context.
func FromContext(ctx context.Context) (token jwt.Claims, ok bool) {
	token, ok = ctx.Value(authKey{}).(jwt.Claims)
	return
}

func authIsNotOK(auths []string) bool {
	return len(auths) != 2 || !strings.EqualFold(auths[0], BearerWord)
}

// GetUserId 从 context 里读出 UserId，未写入时返回 0。
func GetUserId(ctx context.Context) int64 {
	u, _ := ctx.Value(UserId).(int64)
	return u
}
