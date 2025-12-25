package sensitive

import (
	"errors"
	"gitee.com/aniwar2/musae/gamelib/sensitive"
	"gitee.com/aniwar2/musae/logger"
	"strings"
	"unicode"
)

// CheckSpecialLetters
//
//	@Description: 特殊字符检查
//	@receiver s
//	@param str 给定的校验字符串
//	@param ignoreSpace 是否忽略空白符
//	@return bool 存在特殊字符则返回true
func CheckSpecialLetters(str string, ignoreSpace bool) bool {
	for _, v := range str {
		// 不可显示字符
		if !unicode.IsGraphic(v) {
			return true
		}
		// 空白符
		if !ignoreSpace && unicode.IsSpace(v) {
			return true
		}
		// 掩码
		if unicode.IsMark(v) {
			return true
		}
		// 符号
		if unicode.IsSymbol(v) {
			return true
		}
	}
	// 配置的字符
	letters := []string{} /*strings.Split(excel.GetConfigMgr().GetCfg().NAME_SHIELD, ",")*/
	return strings.ContainsAny(str, strings.Join(letters, ""))
}

// CheckSensitiveWord
//
//	@Description: 屏蔽词校验接口
//	@receiver s
//	@param ctype 校验类型 1=玩家昵称，2=编队名称
//	@param content 待校验内容
//	@return bool 通过返回true，否则返回false
//	@return error
func CheckSensitiveWord(ctype int32, content string) (bool, error) {
	logger.Debugf("CheckSensitiveWord: ctype=%v content=%v", ctype, content)

	// 空的？扔回业务层自己处理
	if content == "" {
		return false, errors.New("content is empty")
	}

	if CheckLocal(content) {
		logger.Debugf("CheckLocal 判定非法 %s", content)
		return false, nil
	}

	return true, nil
}

// 本地屏蔽词接口
func CheckLocal(content string) bool {
	ok, _ := sensitive.GetSensitiveWordMgr().FindIn(content)
	return ok
}
