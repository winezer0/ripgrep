# ripgrep

`ripgrep` 是可作为 CLI 使用、也可由外部 Go 工具导入的 PCRE2 文件搜索库。
其目录遍历、ignore 层级和输出结构参考 `go-ripgrep`，文本模式全部由
`go-pcre2-lite` 执行。

## 环境要求

- Go 1.26+
- `CGO_ENABLED=1`
- 可用的 C 编译器

PCRE2 C 源码由依赖包内嵌，不要求系统预装 PCRE2 动态库。

## CLI

```powershell
$env:GOWORK='off'
go build -buildvcs=false -o ripgrep.exe ./cmd/ripgrep
./ripgrep.exe '(?<=user:)\w+' .
```

支持常用 ripgrep 参数，包括递归搜索、ignore、glob、文件类型、上下文、替换、
JSON、颜色、排序、压缩文件和并发线程。退出码与 ripgrep 一致：有匹配为 `0`，
无匹配为 `1`，参数或运行错误为 `2`。

当前兼容目标是 `go-ripgrep v0.0.5` 已实现的参数集合，并非 Rust ripgrep 全部参数。

## 高层 API

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/winezer0/ripgrep"
)

func main() {
	results, err := ripgrep.Search(context.Background(), []string{"."}, ripgrep.Options{
		Pattern: `(?<=TODO:)\s+\K.+`,
		Globs:   []string{"*.go", "!vendor/**"},
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, result := range results {
		fmt.Printf("%s: %d\n", result.Path, result.Stats.Matches)
	}
}
```

需要边搜索边消费时使用 `SearchStream`，并同时读取 `Results` 与 `Errors`。

## 分层 API

- `pkg/matcher`：PCRE2 编译、匹配、字节范围和替换。
- `pkg/searcher`：搜索 `io.Reader`、普通文件和压缩文件。
- `github.com/winezer0/go-ignore`：Git ignore 规则、目录层级和诊断。
- `pkg/ignore`：CLI 文件类型过滤。
- `pkg/globset`：CLI `-g/--glob` 过滤。
- `pkg/printer`：文本、颜色和 NDJSON 输出。

### Ignore 诊断

```go
import (
	"fmt"
	"log"

	ignore "github.com/winezer0/gitignore"
)

func explainIgnore() {
	rules, err := ignore.CompileFile(".gitignore")
	if err != nil {
		log.Fatal(err)
	}
	ignored, detail := rules.MatchesPathHow("build/output.exe", false)
	if detail != nil {
		fmt.Printf("ignored=%t source=%s line=%d rule=%s\n",
			ignored, detail.SourcePath, detail.Line, detail.Pattern)
	}
}
```

`IgnoreStack.MatchesPathHow` 还会考虑隐藏路径、`.rgignore`、`.ignore`、
`.gitignore`、`.git/info/exclude`、全局 Git ignore 和目录层级优先级。

### 仓库 Ignore API

外部工具可创建一次性规则快照并并发查询任意仓库内路径：

```go
config := ignore.DefaultRepositoryConfig()
config.IgnoreFileNames = []string{
	".gitignore",
	".ignore",
	".rgignore",
	".ripgrepignore",
}

matcher, err := ignore.NewRepositoryMatcherWithConfig("/path/to/repo", config)
if err != nil {
	log.Fatal(err)
}
ignored, detail, err := matcher.MatchesPathHow("src/generated.go", false)
```

`IgnoreFileNames` 按从低到高的优先级排列，默认值为 `.gitignore`、`.ignore`、
`.rgignore`。名称只能是单个文件名；需要加载指定路径的规则文件时使用
`AdditionalIgnoreFiles`，其优先级高于逐目录规则。`RepositoryMatcher` 创建后规则
不再变化；ignore 文件更新后需要重新创建匹配器。路径超出规则根目录时返回错误。

CLI 遍历继续使用 `IgnoreStack` 动态加载目录规则。可通过其 `IgnoreFileNames` 字段
使用相同的自定义文件名和优先级语义。

## 许可证注意事项

派生代码和第三方声明见 `THIRD_PARTY_NOTICES.md`。截至所审查版本，
`go-pcre2-lite` 的 Go 包装代码尚未声明许可证；正式发布源码或二进制前必须完成
许可证确认。
