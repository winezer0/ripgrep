# Implementation Notes

## 2026-07-31

- 使用模块路径 `github.com/winezer0/ripgrep`。
- 按当前开发环境采用 Go `1.26.0` 指令。
- 项目当前为空骨架；`internal/app.Run` 仅返回成功退出码，不实现搜索功能。
- `pkg` 仅保留包文档，待公共 API 明确后再扩展，避免提前引入无依据的接口。
- CLI 入口放在 `cmd/ripgrep/main.go`，业务逻辑后续放入 `internal`。
- 父目录的 `go.work` 未包含本模块，独立验证时使用 `GOWORK=off`，未修改父项目配置。
- 父仓库 VCS 元数据无法读取，构建验证关闭 VCS stamping，不影响生成程序。
- 空 CLI 入口直接调用 `app.Run`，便于单元测试覆盖；功能实现后可按错误处理需求扩展退出码策略。

## 2026-07-31 ripgrep 复刻

- 参考 `github.com/startvibecoding/go-ripgrep` 默认分支提交 `54dc948d2bcb9f11a8b8a6e2c60d8ffaa79720b8`（MIT）。
- 参考 `github.com/VillanCh/go-pcre2-lite` 默认分支提交 `c123dedcb5fbdf47fd7e44bad2f7dceeaff48748`，依赖发布版 `v0.1.2`。
- “尽量完整兼容”以所选 `go-ripgrep v0.0.5` 已实现的 CLI 行为为第一兼容基线，再补充 PCRE2 语法；它不是 Rust ripgrep 全部参数的完成声明。
- 所有文本模式（包括 `--fixed-strings` 转义后的模式）均由 PCRE2 执行，不使用 Go RE2 作为文本匹配后端。
- PCRE2 使用 UTF/UCP 和 UTF-8 字节偏移，匹配运行错误会沿调用链返回，不能像参考 SDK 一样静默丢弃。
- `go-pcre2-lite` 要求 `CGO_ENABLED=1` 和可用的 C 编译器，因此本项目不再具备纯 Go 静态交叉编译特性。
- `go-pcre2-lite` 仓库说明其 Go 包装代码尚未选择许可证；直接分发依赖源码或含该代码的二进制前，应由项目所有者完成许可证确认。内嵌 PCRE2 C 源采用 BSD 风格 PCRE2 许可证。
- 参考项目的现有测试覆盖率不足 90%，测试会按本仓库规范扩充，不以其覆盖率声明作为验收依据。

## 2026-07-31 ignore 语义增强

- 参考 `github.com/sabhiram/go-gitignore` 提交 `525f6e181f062064d83887ed2530e3b1ba0bc95a` 的规则说明与测试，但不直接依赖该库；其尾随空格仍有 TODO、`?` 语义有偏差且忽略正则编译错误。
- Git ignore 规则与 CLI `-g/--glob` 使用不同语义，因此新增专用 `RuleSet`，保留 `pkg/globset` 只处理命令行过滤。
- `RuleSet` 负责单个 ignore 文件内的顺序、否定、转义、目录规则和 `**`；`IgnoreStack` 继续负责 `.rgignore > .ignore > .gitignore` 及目录层级。
- `MatchesPathHow` 返回最终规则的来源文件、1-based 行号、规则原文、否定状态和最终结果。
- 遵循 Git 限制：父目录已被某规则排除时，子路径不能通过否定规则重新包含。
- `IsIgnored` 现返回 `(bool, error)`；这是为避免吞掉跨卷 `filepath.Rel` 错误而做出的公开 API 变更。
- 已支持注释与转义前缀、未转义尾随空格、`?`、字符类、根锚定、目录规则、`**`、顺序否定、UTF-8 BOM、`.git/info/exclude` 和全局 ignore。
- `.git` 为 worktree 指针文件时不会错误读取 `.git/info/exclude`；当前尚未解析指针文件指向的实际 gitdir，因此该场景的 info/exclude 不加载。
- `MatchesPathHow` 当前作为 Go API 提供，尚未增加 CLI `--debug/--trace` 展示参数。
- 验证结果：全量测试、竞态检测、`go vet`、CGO 构建和 PCRE2 lookbehind CLI 搜索均通过；各含代码包覆盖率为 74.6%–93.8%，满足调整后的 70% 门槛。

## 2026-07-31 ignore 仓库 API 增强

- 保留 `IgnoreStack` 作为 CLI 遍历时的动态规则栈，新增不可变的 `RepositoryMatcher` 供 codeindex、IDE 和其他外部工具进行随机路径查询；本次不修改 codeindex。
- `RepositoryMatcher` 在构造时预扫描规则文件，构造完成后仅执行只读查询，可由多个 goroutine 并发使用；规则文件变化后需要重新构造。
- `.gitignore`、`.ignore`、`.rgignore` 不再作为三个固定字段保存。`IgnoreFileNames` 按数组顺序从低到高定义每层目录的规则来源，默认顺序保持为 `[]string{".gitignore", ".ignore", ".rgignore"}`。
- 自定义逐目录规则名称只允许单个文件名，避免配置绕过目录层级；指定路径的额外规则通过 `AdditionalIgnoreFiles` 加载，并具有高于逐目录规则的优先级。
- 路径边界使用 `filepath.Rel` 判断并拒绝 `..` 越界，不使用可能误判同前缀兄弟目录的字符串前缀比较。该检查是词法边界检查，不要求待匹配路径已存在。
- 匹配时从规则根目录到目标逐级判断目录；任一父目录被排除后立即停止，因此更深层 ignore 文件不能重新包含其内容。
- 只有位于完整路径组件边界的恰好两个星号才采用 Git globstar 语义；其他连续星号按单路径组件通配符处理。字符类解析支持 Go regexp 提供的 POSIX 类，如 `[[:digit:]]`。
- 本轮验证通过全项目 `go test -cover ./...`、`go test -race ./...`、`go vet ./...` 和 CLI 构建；`pkg/ignore` 覆盖率为 84.2%，所有 Go 文件均未超过 400 行。

## 2026-07-31 ignore 模块拆分

- ignore 核心已移动到独立模块 `github.com/winezer0/go-ignore`，本地目录为同级 `../go-ignore`；ripgrep 通过 `replace` 在发布前引用本地模块。
- ripgrep 的 `pkg/ignore` 仅保留 `types.go` 文件类型过滤，遍历器分别导入公共 ignore 模块和本地类型过滤包。
- 独立模块采用 Go 1.20、MIT 许可证且仅依赖标准库；本次未修改 codeindex。
- 根包新增 `doc.go`，记录阻塞/流式搜索入口、通道消费要求、结果字节偏移、文件发现行为和 CGO 构建条件。

## 2026-07-31 官方 ripgrep 差异文档

- 以 Rust 官方 `BurntSushi/ripgrep 15.2.0` 为基线；该版本发布于 2026-07-15，是核查日 GitHub API 返回的最新稳定版。
- 对照官方 tag `15.2.0` 的 README、GUIDE、FAQ 和 `crates/core/flags/defs.rs`，同时核查本地 CLI parser、walker、matcher、searcher 和 printer，而非仅比较帮助文本。
- 差异文档记录功能、同名参数语义、安装部署、命令迁移和选型建议；未进行同环境性能基准，因此只说明架构差异，不给出未经测量的性能数字。
- 已验证本地全量测试；差异文档共 104 个官方逻辑参数与 36 个 ripgrep 帮助入口的统计均直接来自对应源码定义，未把别名和否定形式重复计数。
