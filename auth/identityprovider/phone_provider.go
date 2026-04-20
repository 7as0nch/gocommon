package identityprovider

import (
	"context"
	"regexp"

	"github.com/7as0nch/gocommon/enums"
)

// 简版严格手机号校验：大陆 11 位手机号，首位为 1，第二位为 3-9。
var phoneStrictRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)

// IsValidPhoneStrict 对外导出，方便业务层复用一致的校验规则。
func IsValidPhoneStrict(phone string) bool {
	return phoneStrictRegex.MatchString(phone)
}

// SMSCaptchaVerifier 手机验证码校验器，由业务方按自身缓存/服务实现。
type SMSCaptchaVerifier interface {
	VerifySMSCaptcha(ctx context.Context, phone, code string) error
}

type PhoneProvider struct {
	verifier SMSCaptchaVerifier
}

func NewPhoneProvider(verifier SMSCaptchaVerifier) *PhoneProvider {
	return &PhoneProvider{verifier: verifier}
}

func (p *PhoneProvider) Type() enums.IdentityType {
	return enums.IdentityTypePhone
}

func (p *PhoneProvider) PrepareRegister(ctx context.Context, input RegisterInput) (*RegisterResult, error) {
	if !IsValidPhoneStrict(input.Identifier) {
		return nil, ErrInvalidPhoneFormat
	}
	if err := p.verifySMSCaptcha(ctx, input.Identifier, input.Credential); err != nil {
		return nil, err
	}
	return &RegisterResult{
		Identifier:     input.Identifier,
		Phone:          input.Identifier,
		AuthCredential: input.Credential,
	}, nil
}

func (p *PhoneProvider) VerifyLogin(ctx context.Context, input LoginInput, _ User, _ UserAuth) error {
	if !IsValidPhoneStrict(input.Identifier) {
		return ErrInvalidPhoneFormat
	}
	return p.verifySMSCaptcha(ctx, input.Identifier, input.Credential)
}

func (p *PhoneProvider) verifySMSCaptcha(ctx context.Context, phone, code string) error {
	if p.verifier == nil {
		return ErrProviderNotConfigured
	}
	return p.verifier.VerifySMSCaptcha(ctx, phone, code)
}
