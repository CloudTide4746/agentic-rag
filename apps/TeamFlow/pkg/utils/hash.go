package utils

import "golang.org/x/crypto/bcrypt"

// 采用 bcrypt 加密
const bcryptCost = 10

// HashPassword 对明文密码进行 bcrypt 哈希。
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword 校验明文密码是否与哈希值匹配。
func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
