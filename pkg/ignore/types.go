package ignore

import (
	"path/filepath"
	"strings"
)

// BuiltInTypes 将文件类型名称映射到 glob 列表。
var BuiltInTypes = map[string][]string{
	"asm":      {"*.asm", "*.s", "*.S"},
	"c":        {"*.c", "*.h"},
	"cpp":      {"*.cpp", "*.cc", "*.cxx", "*.c++", "*.h", "*.hpp", "*.hxx", "*.h++"},
	"css":      {"*.css"},
	"go":       {"*.go"},
	"html":     {"*.html", "*.htm", "*.xhtml"},
	"java":     {"*.java", "*.jsp"},
	"js":       {"*.js", "*.jsx", "*.mjs", "*.cjs"},
	"json":     {"*.json", "*.ipynb"},
	"markdown": {"*.md", "*.markdown", "*.mdown", "*.mkdn"},
	"python":   {"*.py", "*.pyi"},
	"rust":     {"*.rs"},
	"ts":       {"*.ts", "*.tsx", "*.mts", "*.cts"},
	"yaml":     {"*.yaml", "*.yml"},
}

// MatchType 判断文件名是否匹配指定内置类型，并返回 glob 解析错误。
func MatchType(filename string, typeName string) (bool, error) {
	globs, ok := BuiltInTypes[strings.ToLower(typeName)]
	if !ok {
		return false, nil
	}
	for _, glob := range globs {
		matched, err := filepath.Match(glob, filename)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

// ShouldIgnoreByType 根据类型过滤器判断是否排除文件，并返回 glob 解析错误。
func ShouldIgnoreByType(filename string, types []string, typesNot []string) (bool, error) {
	// If positive type filters are specified, file must match at least one
	if len(types) > 0 {
		matchedAny := false
		for _, t := range types {
			matched, err := MatchType(filename, t)
			if err != nil {
				return false, err
			}
			if matched {
				matchedAny = true
				break
			}
		}
		if !matchedAny {
			return true, nil
		}
	}

	// If negative type filters are specified, file must NOT match any
	if len(typesNot) > 0 {
		for _, t := range typesNot {
			matched, err := MatchType(filename, t)
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
		}
	}

	return false, nil
}
