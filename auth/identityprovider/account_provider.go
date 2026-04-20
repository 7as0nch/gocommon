package identityprovider

import (
	"context"
	"errors"
	"log"
	"regexp"

	"golang.org/x/crypto/bcrypt"

	"github.com/7as0nch/gocommon/enums"
)

// DefaultPasswordRegex 默认密码规则：8-32 位，包含至少一个字母和一个数字。
const DefaultPasswordRegex = `^(?=.*[A-Za-z])(?=.*\d)[A-Za-z\d@$!%*?&._-]{8,32}$`

// AccountProvider 账号密码登录，密码规则通过构造函数注入，避免依赖应用字典。
type AccountProvider struct {
	passwordRegex string
}

func NewAccountProvider(passwordRegex string) *AccountProvider {
	if passwordRegex == "" {
		passwordRegex = DefaultPasswordRegex
	}
	return &AccountProvider{passwordRegex: passwordRegex}
}

func (p *AccountProvider) Type() enums.IdentityType {
	return enums.IdentityTypeAccount
}

func (p *AccountProvider) PrepareRegister(_ context.Context, input RegisterInput) (*RegisterResult, error) {
	if err := validatePasswordByPattern(input.Credential, p.passwordRegex); err != nil {
		return nil, err
	}
	hashed, err := HashPassword(input.Credential)
	if err != nil {
		return nil, err
	}
	return &RegisterResult{
		Identifier: input.Identifier,
		Password:   hashed,
	}, nil
}

func (p *AccountProvider) VerifyLogin(_ context.Context, input LoginInput, user User, _ UserAuth) error {
	if user == nil {
		return ErrInvalidCredential
	}
	if !CheckPassword(input.Credential, user.GetPassword()) {
		return ErrInvalidCredential
	}
	return nil
}

// ValidatePassword 按调用方提供的正则校验密码，pattern 为空时使用默认规则。
func ValidatePassword(password, pattern string) error {
	if pattern == "" {
		pattern = DefaultPasswordRegex
	}
	return validatePasswordByPattern(password, pattern)
}

func validatePasswordByPattern(password, pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Printf("密码规则配置错误: %v", err)
		return ErrPasswordRuleMisconfig
	}
	if !re.MatchString(password) {
		return ErrInvalidPasswordRule
	}
	return nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", errors.Join(ErrPasswordEncrypt, err)
	}
	return string(bytes), nil
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
