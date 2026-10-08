package validator

import (
	"fmt"
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
)

// Trans 全局翻译器
var Trans ut.Translator

// Init 初始化校验器：注册中文翻译器和自定义规则
func Init() error {
	// 1. 获取 Gin 的 validator 实例
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return fmt.Errorf("获取 validator 实例失败")
	}

	// 2. 创建中文翻译器
	zhLocale := zh.New()
	uni := ut.New(zhLocale, zhLocale)
	Trans, _ = uni.GetTranslator("zh")

	// 3. 注册中文翻译（内置规则的中文提示）
	if err := zhTranslations.RegisterDefaultTranslations(v, Trans); err != nil {
		return fmt.Errorf("注册中文翻译失败: %w", err)
	}

	// 4. 注册自定义校验规则
	if err := v.RegisterValidation("mobile", validateMobile); err != nil {
		return fmt.Errorf("注册 mobile 校验失败: %w", err)
	}

	// 5. 为自定义规则注册中文提示
	_ = v.RegisterTranslation("mobile", Trans, func(ut ut.Translator) error {
		return ut.Add("mobile", "{0}格式不正确，请输入有效的手机号", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("mobile", fe.Field())
		return t
	})

	return nil
}

// validateMobile 自定义：校验中国大陆手机号
func validateMobile(fl validator.FieldLevel) bool {
	matched, _ := regexp.MatchString(`^1[3-9]\d{9}$`, fl.Field().String())
	return matched
}

// TranslateErrors 将 validator.ValidationErrors 转换为字段->错误信息的 map
// 调用方可将此 map 直接序列化后返回给前端
func TranslateErrors(err error) map[string]string {
	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		// 非校验错误（如 JSON 格式错误），直接包装返回
		return map[string]string{"error": err.Error()}
	}

	result := make(map[string]string, len(errs))
	for _, e := range errs {
		// Translate() 使用已注册的中文翻译器翻译错误信息
		result[e.Field()] = e.Translate(Trans)
	}
	return result
}
