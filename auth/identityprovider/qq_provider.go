package identityprovider

import (
	"context"
	"strings"

	"github.com/7as0nch/gocommon/enums"
)

type QQAuthResult struct {
	OpenID string
}

type QQAuthorizer interface {
	Authorize(ctx context.Context, code string) (*QQAuthResult, error)
}

var defaultQQAuthorizer QQAuthorizer

func SetQQAuthorizer(authorizer QQAuthorizer) {
	defaultQQAuthorizer = authorizer
}

type QQProvider struct {
	authorizer QQAuthorizer
}

func NewQQProvider(authorizer QQAuthorizer) *QQProvider {
	return &QQProvider{
		authorizer: authorizer,
	}
}

func (p *QQProvider) Type() enums.IdentityType {
	return enums.IdentityTypeQQ
}

func (p *QQProvider) PrepareRegister(ctx context.Context, input RegisterInput) (*RegisterResult, error) {
	openID, err := p.resolveOpenID(ctx, input.Credential)
	if err != nil {
		return nil, err
	}
	return &RegisterResult{
		Identifier:     openID,
		AuthCredential: input.Credential,
	}, nil
}

func (p *QQProvider) VerifyLogin(ctx context.Context, input LoginInput, _ User, auth UserAuth) error {
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

func (p *QQProvider) ResolveLoginIdentifier(ctx context.Context, input LoginInput) (string, error) {
	return p.resolveOpenID(ctx, input.Credential)
}

func (p *QQProvider) resolveOpenID(ctx context.Context, credential string) (string, error) {
	authorizer := p.authorizer
	if authorizer == nil {
		authorizer = defaultQQAuthorizer
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
