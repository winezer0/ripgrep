// Package gogrep 提供基于 PCRE2 的文件内容搜索 API，并包含与命令行工具相同的
// 目录遍历、ignore、glob、文件类型、上下文、替换和压缩文件搜索能力。
//
// # 搜索入口
//
// Search 执行阻塞式搜索并返回全部结果。paths 为空时搜索当前目录；ctx 取消后搜索
// 会尽快停止。搜索期间产生的多个错误通过 errors.Join 聚合返回。
//
//	results, err := gogrep.Search(ctx, []string{"."}, gogrep.Options{
//		Pattern: `(?i)todo:\s+(.+)`,
//		Globs:   []string{"*.go", "!vendor/**"},
//	})
//
// SearchStream 在结果产生时立即返回。调用方必须持续消费 Results 和 Errors，直到
// 两个通道都关闭；只读取其中一个通道可能阻塞搜索流水线。每次 Search 或
// SearchStream 调用都创建独立的匹配器和工作协程。
//
// # 匹配与结果
//
// Pattern 使用 PCRE2 语法。Options 可启用固定字符串、不区分大小写、整词、反向
// 匹配、替换、上下文、最大匹配数和 PCRE2 限制。FileResult 按文件保存匹配行和
// 统计信息；Submatch 的 Start 与 End 是原始 UTF-8 文本中的字节偏移。
//
// 默认并行度为 runtime.NumCPU。指定排序模式时使用单个工作线程以保持确定性。
// 调用方不应在搜索进行期间修改传入 Options 中的切片。
//
// # 文件发现
//
// 目录搜索遵循 github.com/winezer0/goignore 提供的 Git 风格忽略规则，并支持
// CLI glob、隐藏文件、符号链接、最大深度和内置文件类型过滤。SearchZip 启用后
// 还会搜索受支持的 gzip、bzip2 和 zip 内容。
//
// # 环境要求
//
// 文本匹配后端 github.com/VillanCh/go-pcre2-lite 使用 CGO，因此构建 gogrep
// 需要 CGO_ENABLED=1 和可用的 C 编译器。PCRE2 C 源由依赖模块提供，不要求系统
// 预装 PCRE2 动态库。
package gogrep
