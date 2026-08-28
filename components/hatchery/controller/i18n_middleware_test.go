package controller

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"hatchery/i18n"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func doRequest(key i18n.Key, args []any) string {
	var result string

	var testHandler = func(w http.ResponseWriter, r *http.Request) {
		result = i18n.T(r.Context(), key, args...)
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Accept-Language", "en")
	w := httptest.NewRecorder()

	wrapper := I18nMiddleware(http.HandlerFunc(testHandler))
	wrapper.ServeHTTP(w, req)

	return result
}

// keysFilePath 定位 keys.go 的绝对路径。
func keysFilePath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to locate current test file via runtime.Caller")
	}
	return filepath.Join(filepath.Dir(thisFile), "../i18n/keys.go")
}

// extractMsgKeyField 解析 keys.go 中所有以 Msg 开头的变量，
// 从复合字面量 Key{...} 中提取指定字段的值。
// fieldName: "string"。
// 返回 [(变量名, 字段值), ...]。
func extractMsgKeyField(t *testing.T, fieldName string) [][2]string {
	t.Helper()
	keysPath := keysFilePath(t)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, keysPath, nil, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse %s failed: %v", keysPath, err)
	}

	var result [][2]string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if !strings.HasPrefix(name.Name, "Msg") {
					continue
				}
				if i >= len(vs.Values) {
					continue
				}
				// 复合字面量 Key{string: "..."}
				comp, ok := vs.Values[i].(*ast.CompositeLit)
				if !ok {
					continue
				}
				fieldVal := extractFieldFromCompositeLit(comp, fieldName)
				if fieldVal == "" {
					continue
				}
				result = append(result, [2]string{name.Name, fieldVal})
			}
		}
	}
	return result
}

// extractFieldFromCompositeLit 从 Key{string: "a"} 中提取指定字段的值。
func extractFieldFromCompositeLit(comp *ast.CompositeLit, field string) string {
	for _, elt := range comp.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		keyIdent, ok := kv.Key.(*ast.Ident)
		if !ok || keyIdent.Name != field {
			continue
		}
		valLit, ok := kv.Value.(*ast.BasicLit)
		if !ok || valLit.Kind != token.STRING {
			continue
		}
		s, err := strconv.Unquote(valLit.Value)
		if err != nil {
			return ""
		}
		return s
	}
	return ""
}

// extractMsgConstants 向后兼容的别名：提取 string 字段。
func extractMsgConstants(t *testing.T) [][2]string {
	return extractMsgKeyField(t, "string")
}

// buildArgsFor 根据 format 中的格式动词，构造一组类型匹配的占位参数，
// 避免 fmt 因缺参产生 `%!s(MISSING)` 之类的标记，从而干扰"是否翻译"的判断。
//
// 仅识别常用动词：%s/%q/%v/%d/%x/%o/%b/%c/%t/%f/%g/%e/%p。
// `%%` 是字面百分号，不消费参数。
func buildArgsFor(format string) []any {
	var args []any
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		// 跳过 flags / width / precision，直到动词字符
		j := i + 1
		for j < len(format) {
			c := format[j]
			if (c >= '0' && c <= '9') || c == '-' || c == '+' || c == '#' || c == ' ' || c == '.' || c == '*' {
				j++
				continue
			}
			break
		}
		if j >= len(format) {
			break
		}
		verb := format[j]
		i = j
		switch verb {
		case '%':
			// `%%` 不消费参数
		case 's', 'q', 'v', 'x', 'X':
			args = append(args, "X")
		case 'd', 'o', 'O', 'b', 'c', 'U':
			args = append(args, 0)
		case 't':
			args = append(args, false)
		case 'f', 'F', 'g', 'G', 'e', 'E':
			args = append(args, 0.0)
		case 'p':
			args = append(args, uintptr(0))
		default:
			// 未识别动词，按 %v 兜底
			args = append(args, "X")
		}
	}
	return args
}

func TestI18n_Untranslated(t *testing.T) {
	consts := extractMsgConstants(t)
	if len(consts) == 0 {
		t.Fatalf("no Msg* variables extracted from keys.go")
	}

	var untranslated []string
	for _, kv := range consts {
		name, strVal := kv[0], kv[1]
		key := i18n.NewKey(strVal)
		args := buildArgsFor(strVal)

		// expected 是"用相同参数对原 key 自身做 Sprintf"的结果。
		// 如果英文翻译条目缺失，message.Printer 会回退到 key 本身，
		// 此时 result 必然等于 expected；只要英文条目存在，result 就会不同。
		// 这样可以避免 `%!s(MISSING)` 之类的 fmt 错误标记把"未翻译"误判为"已翻译"。
		expected := fmt.Sprintf(strVal, args...)
		result := doRequest(key, args)

		if result == expected {
			untranslated = append(untranslated, name)
			t.Errorf("variable %s not translated: `%s`", name, strVal)
		}
	}

	t.Logf("total=%d untranslated=%d", len(consts), len(untranslated))
}

// TestI18n_StringUniqueness 确保 keys.go 中所有 Msg* 变量的 string 值均不重复。
func TestI18n_StringUniqueness(t *testing.T) {
	strings := extractMsgKeyField(t, "string")
	if len(strings) == 0 {
		t.Fatalf("no Msg* variables with string extracted from keys.go")
	}

	seen := make(map[string]string, len(strings)) // string value -> first var name
	var duplicates []string

	for _, kv := range strings {
		name, val := kv[0], kv[1]
		if val == "" {
			t.Errorf("variable %s has empty string value", name)
			continue
		}
		if first, exists := seen[val]; exists {
			duplicates = append(duplicates, val)
			t.Errorf("duplicate string value %q: %s and %s", val, first, name)
		} else {
			seen[val] = name
		}
	}

	t.Logf("total=%d unique=%d duplicates=%d", len(strings), len(seen), len(duplicates))
}

func TestSetDefaultLang_CanRestoreChinese(t *testing.T) {
	i18n.SetDefaultLang("en")
	if !i18n.IsOverseas() {
		t.Fatal("SetDefaultLang(en) should switch site to overseas")
	}
	i18n.SetDefaultLang("zh")
	if i18n.IsOverseas() {
		t.Fatal("SetDefaultLang(zh) should restore domestic site")
	}
}
