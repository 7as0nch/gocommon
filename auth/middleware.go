// Package auth
// @author chengjiang
// @note  Kratos 服务端/客户端 JWT 中间件，已升级到 jwt/v5。
package auth

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/golang-jwt/jwt/v5"
)

// Server is a server auth middleware. Check the token and extract the info from token.
func Server(keyFunc jwt.Keyfunc, opts ...Option) middleware.Middleware {
	o := &options{
		signingMethod: jwt.SigningMethodHS256,
	}
	for _, opt := range opts {
		opt(o)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			header, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, ErrWrongContext
			}
			if keyFunc == nil {
				return nil, ErrMissingKeyFunc
			}
			auths := strings.SplitN(header.RequestHeader().Get(AuthorizationKey), " ", 2)
			if authIsNotOK(auths) {
				return nil, ErrMissingJwtToken
			}
			jwtToken := auths[1]

			var (
				tokenInfo *jwt.Token
				err       error
			)
			if o.claims != nil {
				tokenInfo, err = jwt.ParseWithClaims(jwtToken, o.claims(), keyFunc)
			} else {
				tokenInfo, err = jwt.Parse(jwtToken, keyFunc)
			}
			if err != nil {
				// jwt/v5 使用 sentinel error，通过 errors.Is 判断具体错因。
				switch {
				case stderrors.Is(err, jwt.ErrTokenMalformed):
					return nil, ErrTokenInvalid
				case stderrors.Is(err, jwt.ErrTokenExpired), stderrors.Is(err, jwt.ErrTokenNotValidYet):
					return nil, ErrTokenExpired
				case stderrors.Is(err, jwt.ErrTokenSignatureInvalid):
					return nil, ErrTokenInvalid
				case stderrors.Is(err, jwt.ErrTokenUnverifiable):
					return nil, ErrTokenParseFail
				default:
					return nil, errors.Unauthorized(Reason, err.Error())
				}
			}
			if !tokenInfo.Valid {
				return nil, ErrTokenInvalid
			}
			if tokenInfo.Method != o.signingMethod {
				return nil, ErrUnSupportSigningMethod
			}
			ctx = NewContext(ctx, tokenInfo.Claims)
			return handler(ctx, req)
		}
	}
}

// Client is a client auth middleware，在 Header 里带上 Bearer token。
func Client(keyProvider jwt.Keyfunc, opts ...Option) middleware.Middleware {
	claims := jwt.RegisteredClaims{}
	o := &options{
		signingMethod: jwt.SigningMethodHS256,
		claims:        func() jwt.Claims { return claims },
	}
	for _, opt := range opts {
		opt(o)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if keyProvider == nil {
				return nil, ErrNeedTokenProvider
			}
			token := jwt.NewWithClaims(o.signingMethod, o.claims())
			if o.tokenHeader != nil {
				for k, v := range o.tokenHeader {
					token.Header[k] = v
				}
			}
			key, err := keyProvider(token)
			if err != nil {
				return nil, ErrGetKey
			}
			tokenStr, err := token.SignedString(key)
			if err != nil {
				return nil, ErrSignToken
			}
			if clientContext, ok := transport.FromClientContext(ctx); ok {
				clientContext.RequestHeader().Set(AuthorizationKey, fmt.Sprintf(BearerFormat, tokenStr))
				return handler(ctx, req)
			}
			return nil, ErrWrongContext
		}
	}
}
