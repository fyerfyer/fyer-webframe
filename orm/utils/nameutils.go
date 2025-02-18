package utils

import (
	"reflect"
	"strings"
	"unicode"
)

func GetTableName[T any]() string {
	var t T
	tType := reflect.TypeOf(t)
	return CamelToSnake(tType.Name())
}

// CamelToSnake 将驼峰式命名转换为下划线命名
func CamelToSnake(camelStr string) string {
	if camelStr == "" {
		return ""
	}

	var result strings.Builder
	var runes []rune = []rune(camelStr)
	length := len(runes)

	for i := 0; i < length; i++ {
		current := runes[i]
		isUpper := unicode.IsUpper(current)
		isLower := unicode.IsLower(current)

		// 检查下一个字符（如果存在）
		var nextIsLower bool
		var nextIsUpper bool
		if i < length-1 {
			nextIsLower = unicode.IsLower(runes[i+1])
			nextIsUpper = unicode.IsUpper(runes[i+1])
		}

		// 检查前一个字符（如果存在）
		var prevIsUpper bool
		if i > 0 {
			prevIsUpper = unicode.IsUpper(runes[i-1])
		}

		if isUpper {
			// 如果当前是大写字母，需要决定是否添加下划线
			shouldAddUnderscore := false

			// 如果不是第一个字符
			if i > 0 {
				// 如果前一个是小写字母，或者下一个是小写字母，需要加下划线
				if (i < length-1 && nextIsLower) || (!prevIsUpper && prevIsUpper != isUpper) {
					shouldAddUnderscore = true
				}
				// 处理连续大写字母的情况
				if prevIsUpper && i < length-1 && nextIsLower {
					shouldAddUnderscore = true
				}
			}

			// 添加下划线（如果需要）
			if shouldAddUnderscore && result.Len() > 0 && result.String()[result.Len()-1] != '_' {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(current))
		} else if isLower {
			// 如果是小写字母，检查是否需要在前添加下划线
			if i > 0 && prevIsUpper && nextIsUpper && result.Len() > 0 && result.String()[result.Len()-1] != '_' {
				result.WriteRune('_')
			}
			result.WriteRune(current)
		}
	}

	return result.String()
}

//func main() {
//	testCases := []string{
//		"TestModel",
//		"testModel",
//		"TestABCModel",
//		"HTTPHandler",
//		"SimpleXMLParser",
//		"ABC",
//		"ABCTest",
//		"TestAB",
//		"",
//	}
//
//	for _, test := range testCases {
//		result := CamelToSnake(test)
//		fmt.Printf("输入: %s, 输出: %s\n", test, result)
//	}
//}
