package identityprovider

import (
	"context"
	"errors"

	"github.com/7as0nch/gocommon/enums"
)

var (
	ErrInvalidCredential     = errors.New("凭证无效")
	ErrInvalidPhoneFormat    = errors.New("手机号格式不正确")
	ErrInvalidPasswordRule   = errors.New("密码格式不符合要求")
	ErrPasswordEncrypt       = errors.New("密码加密失败")
	ErrPasswordRuleMisconfig = errors.New("密码规则配置错误")
	ErrProviderNotConfigured = errors.New("登录方式未配置")
	ErrIdentifierRequired    = errors.New("唯一标识不能为空")
)

// User 抽象登录用户的只读视图，避免强耦合具体实体。
type User interface {
	GetPassword() string
}

// UserAuth 抽象用户的第三方认证绑定视图。
type UserAuth interface {
	GetIdentifier() string
}

type RegisterInput struct {
	Identifier string
	Credential string
}

type LoginInput struct {
	Identifier string
	Credential string
}

type RegisterResult struct {
	Identifier     string
	Phone          string
	Password       string
	AuthCredential string
}

type Provider interface {
	Type() enums.IdentityType
	PrepareRegister(ctx context.Context, input RegisterInput) (*RegisterResult, error)
	VerifyLogin(ctx context.Context, input LoginInput, user User, auth UserAuth) error
}

// LoginIdentifierResolver allows providers like QQ/WX to derive
// the lookup identifier (e.g. openid) from credential/code.
type LoginIdentifierResolver interface {
	ResolveLoginIdentifier(ctx context.Context, input LoginInput) (string, error)
}
