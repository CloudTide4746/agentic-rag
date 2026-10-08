package utils

import (
	"errors"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

var maxFileSize int64 = 1024 * 1024 * 10 // 10MB

var allowedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".pdf":  true,
	".doc":  true,
	".docx": true,
	".xls":  true,
	".xlsx": true,
	".txt":  true,
	".zip":  true,
}

// IsAllowedExtension 检验文件扩展名是否在允许的列表中
func IsAllowedExtension(ext string) bool {
	return allowedExtensions[ext]
}

// ValidateFile 验证文件是否符合要求
func ValidateFile(header *multipart.FileHeader) (string, error) {
	if header.Size > maxFileSize {
		return "", errors.New("文件大小不能超过最大限制，当前大小" + strconv.FormatInt(header.Size, 10))
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !IsAllowedExtension(ext) {
		return "", errors.New("文件扩展名不允许")
	}
	return ext, nil

}

// GenerateUniqueFileName 生成唯一的文件名
func GenerateUniqueFileName(ext string) string {
	return uuid.New().String() + ext
}

// GetUploadPath 获取上传路径
func GetUploadPath() string {
	return "storage/uploads"
}
