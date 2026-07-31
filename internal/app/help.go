package app

const helpMessage = `gogrep recursively searches files with PCRE2 patterns.

USAGE:
    gogrep [OPTIONS] PATTERN [PATH...]

OPTIONS:
    -i, --ignore-case          Search case insensitively.
    -s, --case-sensitive       Search case sensitively.
    -w, --word-regexp          Match whole words only.
    -F, --fixed-strings        Treat the pattern as a literal string.
    -v, --invert-match         Select non-matching lines.
    -r, --replace TEXT         Replace matches in displayed output.
    -g, --glob GLOB            Include or exclude paths with a glob.
    -t, --type TYPE            Search only the given file type.
    -T, --type-not TYPE        Exclude the given file type.
        --type-list            List supported file types.
    -A, --after-context NUM    Show lines after each match.
    -B, --before-context NUM   Show lines before each match.
    -C, --context NUM          Show lines before and after each match.
    -m, --max-count NUM        Limit matching lines per file.
    -j, --threads NUM          Set worker count.
        --hidden               Search hidden paths.
        --no-ignore            Ignore no ignore files.
    -L, --follow               Follow symbolic links.
        --json                 Print NDJSON output.
        --color WHEN           Use always, never, or auto color.
        --heading              Group output by file.
        --no-heading           Print the file on every line.
    -n, --line-number          Show line numbers.
    -N, --no-line-number       Hide line numbers.
    -H, --with-filename        Show file names.
    -I, --no-filename          Hide file names.
        --column               Show the first match column.
    -o, --only-matching        Print only matching text.
    -c, --count                Print match counts.
    -q, --quiet                Print no matches.
    -z, --search-zip           Search gzip, bzip2, and zip files.
        --sort SORT            Sort by path, modified, size, or none.
        --sortr SORT           Reverse sort results.
        --max-depth NUM        Limit directory traversal depth.
    -h, --help                 Print help.
    -V, --version              Print version.
`
