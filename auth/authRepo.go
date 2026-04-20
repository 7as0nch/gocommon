// Package auth
// @author chengjiang
// @note  基于 kratos + jwt/v5 的通用鉴权仓库实现。
//        历史版本依赖 aichat 项目内的 tools / tools/strings，现已解耦为
//        本库 utils.GetSFID + 内联字符串判空。
//
//        token 存储通过 TokenStore 接口注入（见 redis.TokenStore），
//        达到：JWT 签名/过期校验 + 服务端吊销列表 的双重防线。
package auth

import (
	"context"
	stderrors "errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/golang-jwt/jwt/v5"

	"github.com/7as0nch/gocommon/utils"
)

// defaultSigningKey 默认签名 key；建议业务侧调用 NewAuthRepoWithKey 注入自己的 key。
const defaultSigningKey = "gocommon/jwt/default-signing-key"

// TokenStore 服务端 token 存储接口；由 redis.TokenStore 等实现。
// 传 nil 时 authRepo 仅做 JWT 签名/过期校验，不做服务端吊销检查。
type TokenStore interface {
	Save(ctx context.Context, userID int64, jti string, ttl time.Duration) error
	Exists(ctx context.Context, userID int64, jti string) (bool, error)
	Revoke(ctx context.Context, userID int64, jti string) error
	RevokeAll(ctx context.Context, userID int64) error
}

type authRepo struct {
	signingKey []byte
	// tokenTTL 签发 token 的有效期，零值回退为 24h。
	tokenTTL time.Duration
	// store 若非 nil：签发时写入、校验时核对，实现服务端主动吊销。
	store TokenStore
}

// AuthRepo 对外暴露的 JWT 仓库接口。
type AuthRepo interface {
	CheckToken(ctx context.Context, token string) (*JwtClaims, error)
	GetToken(ctx context.Context) (string, error)
	NewToken(ctx context.Context, userId int64, username, phone string) (string, error)
	// Logout 吊销当前 token；token 为空时从 context 读取。
	Logout(ctx context.Context, token string) error
	// LogoutAll 吊销该用户所有 token。
	LogoutAll(ctx context.Context, userID int64) error
	// Server 返回 kratos 中间件；仅做 token 校验并把用户信息注入 context。
	Server() func(handler middleware.Handler) middleware.Handler
}

// Options 构造参数。
type Options struct {
	// SigningKey JWT HMAC 签名 key；为空时使用默认 key（仅供本地开发）。
	SigningKey string
	// TokenTTL token 有效期；<=0 时回退为 24h。
	TokenTTL time.Duration
	// Store 服务端 token 存储；nil 时退化为纯 JWT 校验。
	Store TokenStore
}

// NewAuthRepo 使用默认 key 创建仓库；仅供快速试用，生产请用 New。
func NewAuthRepo() AuthRepo {
	return New(Options{})
}

// NewAuthRepoWithKey 使用自定义 key 与 TTL 创建仓库（兼容旧调用方）。
// 新代码推荐使用 New(Options{...}) 以便注入 TokenStore。
func NewAuthRepoWithKey(signingKey string, ttl time.Duration) AuthRepo {
	return New(Options{SigningKey: signingKey, TokenTTL: ttl})
}

// New 按 Options 构造仓库，是推荐的构造入口。
func New(opts Options) AuthRepo {
	key := strings.TrimSpace(opts.SigningKey)
	if key == "" {
		key = defaultSigningKey
	}
	ttl := opts.TokenTTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &authRepo{
		signingKey: []byte(key),
		tokenTTL:   ttl,
		store:      opts.Store,
	}
}

// CheckToken 解析并校验 token，返回 JwtClaims。
// 入参可以是裸 token 也可以是 "Bearer xxx" 形式。
// 若注入了 TokenStore，还会检查 jti 是否在服务端吊销列表外。
func (a *authRepo) CheckToken(ctx context.Context, tokenString string) (*JwtClaims, error) {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return nil, ErrMissingJwtToken
	}
	// 兼容 "Bearer xxx" 和裸 token 两种入参。
	jwtToken := tokenString
	if parts := strings.SplitN(tokenString, " ", 2); len(parts) == 2 {
		if !authIsNotOK(parts) {
			jwtToken = parts[1]
		} else {
			return nil, ErrMissingJwtToken
		}
	}

	token, err := jwt.ParseWithClaims(jwtToken, &JwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrUnSupportSigningMethod
		}
		return a.signingKey, nil
	})
	if err != nil {
		// jwt/v5 不再使用 ValidationError，改用 sentinel error。
		switch {
		case stderrors.Is(err, jwt.ErrTokenMalformed):
			return nil, ErrSignToken
		case stderrors.Is(err, jwt.ErrTokenExpired):
			return nil, ErrTokenExpired
		case stderrors.Is(err, jwt.ErrTokenNotValidYet):
			return nil, ErrTokenInvalid
		case stderrors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, ErrTokenInvalid
		default:
			return nil, ErrTokenInvalid
		}
	}
	claims, ok := token.Claims.(*JwtClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	// 服务端吊销校验：jti 不在桶里视为失效（已登出/被踢下线）。
	if a.store != nil && claims.ID != "" {
		exists, err := a.store.Exists(ctx, claims.UserId, claims.ID)
		if err != nil {
			return nil, fmt.Errorf("check token store: %w", err)
		}
		if !exists {
			return nil, ErrTokenExpired
		}
	}

	return claims, nil
}

// GetToken 从 kratos server context 的 Authorization header 里取 token。
func (a *authRepo) GetToken(ctx context.Context) (string, error) {
	var token string
	if header, ok := transport.FromServerContext(ctx); ok {
		token = header.RequestHeader().Get(AuthorizationKey)
	}
	if strings.TrimSpace(token) == "" {
		return "", stderrors.New("in GetToken, token is nil")
	}
	return token, nil
}

// NewToken 生成一个默认 claims 的 token，并在注入 store 时记录到吊销白名单。
func (a *authRepo) NewToken(ctx context.Context, userId int64, username, phone string) (string, error) {
	expiredAt := jwt.NewNumericDate(time.Now().Add(a.tokenTTL))
	jti := strconv.FormatInt(utils.GetSFID(), 10)
	claims := JwtClaims{
		UserId:    userId,
		UserName:  username,
		UserPhone: phone,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: expiredAt,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(a.signingKey)
	if err != nil {
		return "", fmt.Errorf("创建 token 失败: %w", err)
	}
	if a.store != nil {
		if err := a.store.Save(ctx, userId, jti, a.tokenTTL); err != nil {
			return "", fmt.Errorf("持久化 token 失败: %w", err)
		}
	}
	return signed, nil
}

// Logout 吊销指定 token；token 为空时尝试从 context 中取。
// 若未注入 store，则只做格式校验后返回（无副作用）。
func (a *authRepo) Logout(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		var err error
		token, err = a.GetToken(ctx)
		if err != nil {
			return err
		}
	}
	claims, err := a.parseWithoutStore(token)
	if err != nil {
		return err
	}
	if a.store == nil || claims == nil || claims.ID == "" {
		return nil
	}
	return a.store.Revoke(ctx, claims.UserId, claims.ID)
}

// LogoutAll 吊销用户全部 token。
func (a *authRepo) LogoutAll(ctx context.Context, userID int64) error {
	if a.store == nil {
		return nil
	}
	return a.store.RevokeAll(ctx, userID)
}

// parseWithoutStore 只做 JWT 校验，不查吊销表；专供 Logout 使用。
func (a *authRepo) parseWithoutStore(tokenString string) (*JwtClaims, error) {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return nil, ErrMissingJwtToken
	}
	jwtToken := tokenString
	if parts := strings.SplitN(tokenString, " ", 2); len(parts) == 2 {
		if !authIsNotOK(parts) {
			jwtToken = parts[1]
		} else {
			return nil, ErrMissingJwtToken
		}
	}
	token, err := jwt.ParseWithClaims(jwtToken, &JwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrUnSupportSigningMethod
		}
		return a.signingKey, nil
	})
	// Logout 阶段允许已过期的 token 被解析 —— 业务上正好要清理它。
	if err != nil && !stderrors.Is(err, jwt.ErrTokenExpired) {
		return nil, ErrTokenInvalid
	}
	claims, ok := token.Claims.(*JwtClaims)
	if !ok {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

// Server 返回 kratos server 中间件：校验 token + 注入用户信息。
func (a *authRepo) Server() func(handler middleware.Handler) middleware.Handler {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var token string
			if header, ok := transport.FromServerContext(ctx); ok {
				token = header.RequestHeader().Get(AuthorizationKey)
			}
			if strings.TrimSpace(token) == "" {
				return nil, kerrors.New(401, "Token is missing", "Token is missing")
			}
			claims, err := a.CheckToken(ctx, token)
			if err != nil || claims == nil {
				return nil, kerrors.New(401, "Token is invalid or expired", "Token is invalid or expired")
			}
			ctx = context.WithValue(ctx, UserId, claims.UserId)
			ctx = context.WithValue(ctx, UserName, claims.UserName)
			ctx = context.WithValue(ctx, UserPhone, claims.UserPhone)
			return handler(ctx, req)
		}
	}
}

// NewHeaderServer 一个示例中间件：从 header/metadata 里读 ClientID 并写入 context。
func NewHeaderServer() func(handler middleware.Handler) middleware.Handler {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var clientID string
			if md, ok := metadata.FromServerContext(ctx); ok {
				_ = md.Get("U-OrGniZaTiOn") // 业务自定义 metadata，按需使用。
			}
			if header, ok := transport.FromServerContext(ctx); ok {
				clientID = header.RequestHeader().Get("Clientid")
				ctx = context.WithValue(ctx, ClientID, clientID)
			}
			return handler(ctx, req)
		}
	}
}
