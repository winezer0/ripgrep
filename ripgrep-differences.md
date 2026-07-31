# ripgrep 与 ripgrep 的差异

本文对比当前 `ripgrep 0.1.0` 与 Rust 官方
[`BurntSushi/ripgrep 15.2.0`](https://github.com/BurntSushi/ripgrep/releases/tag/15.2.0)。
`15.2.0` 发布于 2026-07-15，是截至 2026-07-31 的最新稳定版。

参考资料：

- [ripgrep 15.2.0 README](https://github.com/BurntSushi/ripgrep/blob/15.2.0/README.md)
- [ripgrep 15.2.0 User Guide](https://github.com/BurntSushi/ripgrep/blob/15.2.0/GUIDE.md)
- [ripgrep 15.2.0 CLI flags](https://github.com/BurntSushi/ripgrep/blob/15.2.0/crates/core/flags/defs.rs)
- [ripgrep 15.2.0 FAQ](https://github.com/BurntSushi/ripgrep/blob/15.2.0/FAQ.md)
- ripgrep 的 `internal/app/help.go`、`options.go`、`walk.go`、`pkg/matcher`、
  `pkg/searcher` 和 `pkg/printer`

## 结论

`ripgrep` 不是 `rg` 的完全兼容实现，也不应作为无条件替换。它实现了常用的递归
搜索参数，并始终使用 PCRE2，适合需要在 Go 程序内嵌搜索、消费结构化结果或使用
独立 ignore API 的场景。

官方 `rg` 的 CLI、输入格式、过滤、输出模式、配置能力和性能优化明显更完整，适合
通用终端搜索。官方 15.2.0 定义了 104 个逻辑参数，当前 ripgrep 帮助中有 36 个
参数入口；参数别名和否定形式未计入该数字。

## 总体定位

| 项目 | ripgrep | 官方 ripgrep |
| --- | --- | --- |
| 主要定位 | 可嵌入 Go 的搜索库与 CLI | 成熟的通用命令行搜索工具 |
| 当前版本 | 0.1.0 | 15.2.0 |
| 公开编程接口 | Go `Search`、`SearchStream` 和分层包 | Rust crates；没有官方 Go API |
| 默认正则引擎 | 始终 PCRE2 | 默认 Rust regex，可选 PCRE2 或自动选择 |
| 构建要求 | Go、CGO、C 编译器 | 官方预编译二进制；源码构建使用 Rust，PCRE2 可选 |
| CLI 兼容程度 | 常用参数子集 | 基准实现 |
| 性能目标 | 尚无与 15.2.0 的正式基准 | 大规模优化的生产级搜索器 |

## 功能差异

### 正则与模式输入

| 能力 | ripgrep | 官方 ripgrep |
| --- | --- | --- |
| 普通正则 | PCRE2 UTF/UCP | 默认 Rust regex |
| 环视、反向引用 | 默认可用 | 需要 `-P` 或 `--engine pcre2/auto` |
| 引擎选择 | 不支持，固定 PCRE2 | `--engine default/pcre2/auto` |
| 固定字符串、整词、大小写 | 支持 `-F`、`-w`、`-i/-s` | 支持，并额外支持 `--smart-case`、`-x` 等 |
| 多模式 | 不支持 `-e`、`-f` | 支持重复 `-e` 和模式文件 `-f` |
| 多行匹配 | 不支持；逐行执行 PCRE2 | `-U/--multiline` 与 `--multiline-dotall` |
| Unicode 开关 | PCRE2 UTF/UCP 固定开启 | 支持 `--no-unicode` 及 PCRE2 Unicode 控制 |
| 正则限制 | Go API 有 MatchLimit/DepthLimit，CLI 未暴露 | 提供 DFA/regex size 等 CLI 限制 |

即使 PCRE2 模式中包含 `\n`，ripgrep 也不能跨行匹配，因为搜索器在送入 PCRE2 前
已经按行切分输入。单行长度目前最多为 10 MiB，超出后返回 scanner 错误；官方
`rg` 没有这个 ripgrep 特有的固定单行上限。

### 文件发现与 ignore

两者默认都跳过隐藏路径、Git ignore 命中的路径和符号链接，并支持 `.gitignore`、
`.ignore`、`.rgignore`、`.git/info/exclude` 及全局 Git ignore。

| 能力 | ripgrep | 官方 ripgrep |
| --- | --- | --- |
| 关闭全部 ignore | `--no-ignore` | `--no-ignore` |
| 分级关闭 ignore | 不支持 | VCS、global、parent、dot、exclude、显式文件可分别关闭 |
| 不受限模式 | 不支持 `-u/-uu/-uuu` | 支持逐级放开 ignore、hidden、binary |
| 额外 ignore 文件 | ripgrep CLI/Search Options 不支持；独立 `go-ignore` API 支持 | `--ignore-file`，可重复使用 |
| ignore 大小写 | 不支持 CLI 选项 | `--ignore-file-case-insensitive` |
| 自定义逐目录文件名 | `go-ignore` API 支持 | CLI 固定自动发现的文件类型 |
| 匹配诊断 | `MatchesPathHow` 返回来源和行号 | `--debug/--trace` 输出内部诊断 |
| 文件系统边界 | 不支持 | `--one-file-system` |
| Git worktree 指针 | 暂未解析实际 gitdir | 官方实现支持更完整的 Git 场景 |

两者的额外 ignore 文件优先级不同：`go-ignore` 的 `AdditionalIgnoreFiles` 高于逐目录
规则；官方 `rg --ignore-file` 低于自动发现的逐目录规则。迁移时不能直接假设结果
相同。

### glob 与文件类型

ripgrep 支持 `-g/--glob`、`-t/--type`、`-T/--type-not` 和 `--type-list`，但存在以下
差距：

- 官方支持 `--iglob`、`--glob-case-insensitive`、`--type-add` 和 `--type-clear`。
- 官方内置文件类型集合远大于 ripgrep 当前的 14 类。
- ripgrep 的 CLI glob 是独立的 Go 正则转换实现；对绝对搜索根、多条正负 glob 和
  部分边界模式，不承诺与官方 globset 完全一致。
- ripgrep 的类型配置只能通过修改 Go 代码扩展，不能在单次 CLI 调用中增加。

### 编码、二进制与输入

| 能力 | ripgrep | 官方 ripgrep |
| --- | --- | --- |
| UTF-8 | 支持，PCRE2 UTF 模式固定开启 | 支持 |
| UTF-16 与其他编码 | 不支持 | `-E/--encoding`，支持 BOM 检测和 WHATWG 编码标签 |
| 原始字节搜索 | 不支持 | `--no-encoding`、`--no-unicode` 等组合 |
| 二进制检测 | 仅检查输入前 1024 字节是否含 NUL | 搜索过程中持续检测 NUL |
| 强制文本/二进制搜索 | 不支持 `-a/--text`、`--binary` | 支持多种二进制处理模式 |
| stdin | 无路径且 stdin 非终端，或路径为 `-` | 支持 `-` 和管道输入 |
| 预处理器 | 不支持 | `--pre` 与 `--pre-glob` |

ripgrep 如果在前 1024 字节发现 NUL，会直接返回空结果；后续位置才出现 NUL 时仍
可能送入 PCRE2。官方 `rg` 会在整个搜索过程中执行二进制策略，并提供覆盖选项。

### 压缩文件与归档

同名的 `-z/--search-zip` 语义并不相同：

- ripgrep 使用 Go 标准库直接搜索 gzip、bzip2，以及 ZIP 归档中的每个文件条目。
- 官方 `rg` 调用 PATH 中的解压程序，支持 gzip、bzip2、xz、LZ4、LZMA、Brotli
  和 Zstd，但明确不把 ZIP、tar 等归档当作目录树搜索。

因此 ZIP 条目搜索是 ripgrep 当前多出的能力；xz、LZ4、LZMA、Brotli 和 Zstd 是
官方 `rg` 当前多出的能力。

### 输出与统计

| 能力 | ripgrep | 官方 ripgrep |
| --- | --- | --- |
| 行号、列号、文件名、heading | 支持基础开关 | 支持，并有更完整的自动模式 |
| 上下文 | `-A/-B/-C` | 支持，并有 separator、passthru 等选项 |
| 仅匹配、计数、静默 | `-o/-c/-q` | 支持，并有 `--count-matches`、`--include-zero` |
| 替换显示 | `-r`，不修改文件 | `-r`，同样不修改文件 |
| JSON Lines | 简化的 begin/match/context/end/summary | 完整协议，非法 UTF-8 使用 base64 bytes |
| 文件清单 | 不支持 | `--files`、`-l`、`--files-without-match` |
| 其他格式 | 不支持 | byte offset、vimgrep、NUL、stats、trim、hyperlink 等 |
| 颜色 | auto/always/never 与固定配色 | 支持细粒度 `--colors` 配置 |

ripgrep 默认显示文件名和行号，只有 heading 根据终端自动切换；官方 `rg` 会根据输入
数量、终端和输出模式动态决定默认格式。因此依赖文本输出的脚本应显式指定
`-H/-I`、`-n/-N` 和 `--heading/--no-heading`。

ripgrep 的 JSON 消息类型借鉴官方格式，但统计字段、非法 UTF-8 表示和部分模式组合
并不完全兼容。不要把它当作官方 JSON 协议的字节级替代品。

### 排序、错误与性能

- 两者的 `--sort` 都会禁用并行搜索。ripgrep 支持 `path/modified/size/none`；官方
  支持 `path/modified/accessed/created/none`。`size` 是 ripgrep 特有值。
- ripgrep 的遍历或单文件搜索出现错误后会取消当前搜索流水线，并通过 Go API 返回
  错误；官方 `rg` 对许多单路径错误会报告后继续，并提供 `--no-messages` 等控制。
- 官方 `rg` 使用 Rust regex 自动机、SIMD、字面量优化、内存映射和成熟的并行遍历。
  ripgrep 使用 `bufio.Scanner` 逐行读取并始终调用 PCRE2。尚未完成同环境基准测试，
  不应宣称性能等价；普通搜索通常应预期官方 `rg` 更快。

## 使用差异

### 安装与发布

官方 `rg` 提供多平台预编译二进制和主流包管理器安装方式，二进制名为 `rg`。
ripgrep 当前主要从源码构建：

```powershell
$env:GOWORK='off'
$env:CGO_ENABLED='1'
go build -buildvcs=false -o ripgrep.exe ./cmd/ripgrep
```

ripgrep 需要 Go、CGO 和 C 编译器。其 PCRE2 依赖内嵌 C 源，无需系统安装 PCRE2；
但 `go-pcre2-lite` Go 包装层的许可证仍需在正式分发前确认。

### 常用命令迁移

以下用法基本对应：

| 目的 | 官方 ripgrep | ripgrep |
| --- | --- | --- |
| 递归搜索 | `rg 'TODO' .` | `ripgrep 'TODO' .` |
| 不区分大小写 | `rg -i 'todo' .` | `ripgrep -i 'todo' .` |
| 固定字符串 | `rg -F 'a[b]' .` | `ripgrep -F 'a[b]' .` |
| glob 过滤 | `rg -g '*.go' -g '!vendor/**' x .` | 参数形式相同 |
| 文件类型 | `rg -t go x .` | 参数形式相同，但类型集合较少 |
| 上下文 | `rg -C 2 x .` | 参数形式相同 |
| JSON | `rg --json x .` | 参数形式相同，协议不完全兼容 |

以下迁移需要调整：

| 官方 ripgrep 用法 | ripgrep 处理方式 |
| --- | --- |
| `rg -P '(?<=user:)\w+' .` | 去掉 `-P`；ripgrep 默认就是 PCRE2 |
| `rg -e foo -e bar .` | CLI 不支持多模式，可按需要改写为 `foo|bar` |
| `rg -f patterns.txt .` | 不支持模式文件 |
| `rg -U 'foo\nbar' .` | 不支持跨行匹配 |
| `rg -uuu pattern .` | 无完全等价项；`--no-ignore --hidden` 仍不会关闭二进制策略 |
| `rg -a pattern .` | 不支持强制文本模式 |
| `rg -E utf-16le pattern .` | 不支持编码转换 |
| `rg --ignore-file rules pattern .` | ripgrep CLI 不支持；独立 `go-ignore` API 可配置 |
| `rg --type-add 'foo:*.foo' -t foo x .` | CLI 不支持动态类型 |
| `rg --files` 或 `rg -l pattern` | 不支持文件清单输出模式 |
| `rg --pre command pattern .` | 不支持预处理器 |
| `rg --sort accessed/created` | 不支持；ripgrep 额外支持 `--sort size` |

### 配置与集成

官方 `rg` 支持 `RIPGREP_CONFIG_PATH`、`--no-config`、shell completion 和 man page
生成。ripgrep 当前没有配置文件或 completion 机制，所有 CLI 配置必须随命令传入。
两者的搜索退出码约定相同：找到匹配为 0，没有匹配为 1，参数或运行错误为 2。

ripgrep 的主要使用优势是 Go 集成：

```go
results, err := ripgrep.Search(ctx, []string{"."}, ripgrep.Options{
	Pattern: `(?<=TODO:)\s+\K.+`,
	Globs:   []string{"*.go", "!vendor/**"},
})
```

需要边搜索边处理时可使用 `SearchStream`，但必须同时持续消费 `Results` 和
`Errors`，直到两个通道关闭。官方 `rg` 的典型集成方式是启动子进程并解析文本或
JSON Lines；其内部 Rust crates 不提供官方 Go API。

## 选择建议

选择 ripgrep：

- 需要把搜索直接嵌入 Go 进程。
- 需要 Go 结构体结果、取消上下文或流式结果通道。
- 愿意同时使用独立 `go-ignore` 包获取结构化 ignore 诊断和自定义规则文件名。
- 需要直接搜索 ZIP 归档条目。
- 搜索模式依赖 PCRE2，并接受 CGO 构建条件。

选择官方 `rg`：

- 需要完整 CLI、配置文件、completion 或脚本兼容性。
- 需要多模式、多行、编码转换、二进制控制、预处理器或文件清单模式。
- 需要更广泛的压缩格式、文件类型和输出协议。
- 优先考虑成熟度、预编译发布、跨平台部署和经过长期优化的性能。

如果目标是替换已有 `rg` 脚本，应逐项检查参数和输出格式，不能只把命令名从
`rg` 改成 `ripgrep`。
