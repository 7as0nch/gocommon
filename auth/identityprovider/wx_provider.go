package identityprovider

import (
	"context"
	"strings"

	"github.com/7as0nch/gocommon/enums"
)

type WechatAuthResult struct {
	OpenID string
}

type WechatAuthorizer interface {
	Authorize(ctx context.Context, code string) (*WechatAuthResult, error)
}

var defaultWechatAuthorizer WechatAuthorizer

func SetWechatAuthorizer(authorizer WechatAuthorizer) {
	defaultWechatAuthorizer = authorizer
}

type WechatProvider struct {
	authorizer WechatAuthorizer
}

func NewWechatProvider(authorizer WechatAuthorizer) *WechatProvider {
	return &WechatProvider{
		authorizer: authorizer,
	}
}

func (p *WechatProvider) Type() enums.IdentityType {
	return enums.IdentityTypeWechat
}

func (p *WechatProvider) PrepareRegister(ctx context.Context, input RegisterInput) (*RegisterResult, error) {
	openID, err := p.resolveOpenID(ctx, input.Credential)
	if err != nil {
		return nil, err
	}
	return &RegisterResult{
		Identifier:     openID,
		AuthCredential: input.Credential,
	}, nil
}

func (p *WechatProvider) VerifyLogin(ctx context.Context, input LoginInput, _ User, auth UserAuth) error {
	openID, err := p.resolveOpenID(ctx, input.Credential)
	if err != nil {
		return err
	}
	if auth == nil || strings.TrimSpace(auth.GetIdentifier()) == "" {
		return ErrInvalidCredential
	}
	if openID != auth.GetIdentifier() {
		return ErrInvalidCredential
	}
	return nil
}

func (p *WechatProvider) ResolveLoginIdentifier(ctx context.Context, input LoginInput) (string, error) {
	return p.resolveOpenID(ctx, input.Credential)
}

func (p *WechatProvider) resolveOpenID(ctx context.Context, credential string) (string, error) {
	authorizer := p.authorizer
	if authorizer == nil {
		authorizer = defaultWechatAuthorizer
	}
	if authorizer == nil {
		return "", ErrProviderNotConfigured
	}
	result, err := authorizer.Authorize(ctx, credential)
	if err != nil {
		return "", err
	}
	openID := strings.TrimSpace(result.OpenID)
	if openID == "" {
		return "", ErrInvalidCredential
	}
	return openID, nil
}
