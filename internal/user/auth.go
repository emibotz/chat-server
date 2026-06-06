package user

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidFormat       = fmt.Errorf("invalid auth format")
	ErrVersionIncompatible = fmt.Errorf("incompatible argon2 version")
	ErrNotMatch            = fmt.Errorf("password not match")
)

// 认证配置
type AuthConfig struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8

	SaltLength uint32
	KeyLength  uint32
}

func DefaultConfig() *AuthConfig {
	return &AuthConfig{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,

		SaltLength: 16,
		KeyLength:  32,
	}
}

// 从环境变量中加载配置
func (c *AuthConfig) Load() {
	if e := os.Getenv("ARGON2_MEMORY"); e != "" {
		if val, err := strconv.ParseUint(e, 10, 32); err == nil {
			c.Memory = uint32(val)
		}
	}
	if e := os.Getenv("ARGON2_ITERATION"); e != "" {
		if val, err := strconv.ParseUint(e, 10, 32); err == nil {
			c.Iterations = uint32(val)
		}
	}
	if e := os.Getenv("ARGON2_PARALLELISM"); e != "" {
		if val, err := strconv.ParseUint(e, 10, 32); err == nil {
			c.Parallelism = uint8(val)
		}
	}
	if e := os.Getenv("ARGON2_SALT_LENGTH"); e != "" {
		if val, err := strconv.ParseUint(e, 10, 32); err == nil {
			c.SaltLength = uint32(val)
		}
	}
	if e := os.Getenv("ARGON2_KEY_LENGTH"); e != "" {
		if val, err := strconv.ParseUint(e, 10, 32); err == nil {
			c.KeyLength = uint32(val)
		}
	}
}

var Config = DefaultConfig()

var Pepper = ""

// 生成撒上胡椒粉的密码串
func GeneratePepperedPassword(pepper string, password string) []byte {
	h := hmac.New(sha256.New, []byte(pepper))
	h.Write([]byte(password))
	return h.Sum(nil)
}

// 生成可存储的认证串
func GenerateAuth(config *AuthConfig, password string) (string, error) {
	// 生成盐
	salt := make([]byte, config.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// 撒胡椒粉
	input := GeneratePepperedPassword(Pepper, password)

	// 撒盐
	hash := argon2.IDKey(input, salt, config.Iterations, config.Memory, config.Parallelism, config.KeyLength)

	// 生成认证串
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	auth := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, config.Memory, config.Iterations, config.Parallelism, b64Salt, b64Hash,
	)

	return auth, nil
}

// 校验密码和认证串是否匹配
func VerifyPasswordWithAuth(password string, auth string) error {
	// 解析认证串
	parts := strings.Split(auth, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return ErrInvalidFormat
	}
	if parts[2] != "v="+strconv.FormatUint(argon2.Version, 10) {
		return ErrVersionIncompatible
	}

	config := DefaultConfig()
	if _, err := fmt.Sscanf(
		parts[3], "m=%d,t=%d,p=%d",
		&config.Memory, &config.Iterations, &config.Parallelism,
	); err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return fmt.Errorf("decode salt failed: %w", err)
	}
	config.SaltLength = uint32(len(salt))

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return fmt.Errorf("decode key failed: %w", err)
	}
	config.SaltLength = uint32(len(hash))

	// 使用相同的参数对密码进行加密
	input := GeneratePepperedPassword(Pepper, password)
	comparisonHash := argon2.IDKey(input, salt, config.Iterations, config.Memory, config.Parallelism, config.KeyLength)

	// Gemini Suggested: 使用 constant-time 比较，防止计时攻击
	if subtle.ConstantTimeCompare(hash, comparisonHash) == 0 {
		return ErrNotMatch
	}

	return nil
}
