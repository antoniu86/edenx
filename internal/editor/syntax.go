package editor

import (
	"path/filepath"
	"regexp"
	"strings"
)

// SynToken is the syntax token type assigned to each byte position in a line.
type SynToken int

const (
	SynPlain   SynToken = iota
	SynKeyword          // language keywords (if, for, func …)
	SynType             // built-in type names (int, string, bool …)
	SynBuiltin          // built-in functions / constants (len, nil, true …)
	SynString           // string / character literals
	SynComment          // line and inline block comments
	SynNumber           // numeric literals
)

type synRule struct {
	re  *regexp.Regexp
	tok SynToken
}

// synDef holds the ordered rules for one language.
// Rules are applied in order; once a byte is assigned a non-Plain token
// earlier rules win (higher-priority rules should appear first).
type synDef struct {
	rules []synRule
}

// tokenizeLine returns a []SynToken of length len(line), one entry per byte.
func tokenizeLine(line string, def *synDef) []SynToken {
	out := make([]SynToken, len(line))
	for _, rule := range def.rules {
		for _, m := range rule.re.FindAllStringIndex(line, -1) {
			for i := m[0]; i < m[1]; i++ {
				if out[i] == SynPlain {
					out[i] = rule.tok
				}
			}
		}
	}
	return out
}

// synDefForFile returns the language definition for the given filename,
// or nil for plain text / unknown types.
func synDefForFile(filename string) *synDef {
	base := strings.ToLower(filepath.Base(filename))
	if base == "makefile" || base == "gnumakefile" {
		return syntaxDefs[".mk"]
	}
	return syntaxDefs[strings.ToLower(filepath.Ext(filename))]
}

// syntaxDefs maps file extension → language definition.
var syntaxDefs = map[string]*synDef{}

// rule is a convenience constructor.
func rule(pattern string, tok SynToken) synRule {
	return synRule{re: regexp.MustCompile(pattern), tok: tok}
}

func init() {
	// \x60 = backtick — used in patterns that need it but live in double-quoted strings.
	bt := "\x60"

	// ── Go ──────────────────────────────────────────────────────────────────
	syntaxDefs[".go"] = &synDef{rules: []synRule{
		rule(`//.*`, SynComment),
		rule(`/\*.*?\*/`, SynComment),
		rule(bt+`[^`+bt+`]*`+bt, SynString), // raw string literals
		rule(`"(?:[^"\\]|\\.)*"`, SynString),
		rule(`'(?:[^'\\]|\\.)*'`, SynString),
		rule(`\b(0x[0-9a-fA-F]+|0b[01]+|\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)\b`, SynNumber),
		rule(`\b(break|case|chan|const|continue|default|defer|else|fallthrough|for|func|go|goto|if|import|interface|map|package|range|return|select|struct|switch|type|var)\b`, SynKeyword),
		rule(`\b(bool|byte|complex64|complex128|error|float32|float64|int|int8|int16|int32|int64|rune|string|uint|uint8|uint16|uint32|uint64|uintptr|any)\b`, SynType),
		rule(`\b(append|cap|close|complex|copy|delete|imag|len|make|new|panic|print|println|real|recover|true|false|nil|iota)\b`, SynBuiltin),
	}}

	// ── Python ──────────────────────────────────────────────────────────────
	syntaxDefs[".py"] = &synDef{rules: []synRule{
		rule(`#.*`, SynComment),
		rule(`""".*?"""`, SynString),
		rule(`'''.*?'''`, SynString),
		rule(`"(?:[^"\\]|\\.)*"`, SynString),
		rule(`'(?:[^'\\]|\\.)*'`, SynString),
		rule(`\b(0x[0-9a-fA-F]+|\d+(?:\.\d+)?(?:[eE][+-]?\d+)?j?)\b`, SynNumber),
		rule(`\b(and|as|assert|async|await|break|class|continue|def|del|elif|else|except|finally|for|from|global|if|import|in|is|lambda|nonlocal|not|or|pass|raise|return|try|while|with|yield)\b`, SynKeyword),
		rule(`\b(bool|bytes|complex|dict|float|frozenset|int|list|memoryview|object|range|set|str|tuple|type)\b`, SynType),
		rule(`\b(abs|all|any|bin|callable|chr|dir|divmod|enumerate|eval|exec|filter|format|getattr|globals|hasattr|hash|help|hex|id|input|isinstance|issubclass|iter|len|locals|map|max|min|next|oct|open|ord|pow|print|repr|reversed|round|setattr|sorted|sum|super|vars|zip|None|True|False)\b`, SynBuiltin),
	}}

	// ── JavaScript / TypeScript ──────────────────────────────────────────────
	for _, ext := range []string{".js", ".ts", ".jsx", ".tsx", ".mjs"} {
		syntaxDefs[ext] = &synDef{rules: []synRule{
			rule(`//.*`, SynComment),
			rule(`/\*.*?\*/`, SynComment),
			rule(bt+`[^`+bt+`]*`+bt, SynString), // template literals
			rule(`"(?:[^"\\]|\\.)*"`, SynString),
			rule(`'(?:[^'\\]|\\.)*'`, SynString),
			rule(`\b(0x[0-9a-fA-F]+|\d+(?:\.\d+)?(?:[eE][+-]?\d+)?n?)\b`, SynNumber),
			rule(`\b(async|await|break|case|catch|class|const|continue|debugger|default|delete|do|else|export|extends|finally|for|from|function|if|import|in|instanceof|let|new|of|return|static|super|switch|throw|try|typeof|var|void|while|with|yield)\b`, SynKeyword),
			rule(`\b(Array|Boolean|Date|Error|Function|Map|Number|Object|Promise|RegExp|Set|String|Symbol|WeakMap|WeakSet)\b`, SynType),
			rule(`\b(console|document|window|undefined|null|true|false|NaN|Infinity|Math|JSON|parseInt|parseFloat|isNaN|isFinite)\b`, SynBuiltin),
		}}
	}

	// ── JSON ─────────────────────────────────────────────────────────────────
	syntaxDefs[".json"] = &synDef{rules: []synRule{
		rule(`"(?:[^"\\]|\\.)*"`, SynString),
		rule(`\b\d+(?:\.\d+)?(?:[eE][+-]?\d+)?\b`, SynNumber),
		rule(`\b(true|false|null)\b`, SynKeyword),
	}}

	// ── YAML ─────────────────────────────────────────────────────────────────
	for _, ext := range []string{".yml", ".yaml"} {
		syntaxDefs[ext] = &synDef{rules: []synRule{
			rule(`#.*`, SynComment),
			rule(`"(?:[^"\\]|\\.)*"`, SynString),
			rule(`'[^']*'`, SynString),
			rule(`\b\d+(?:\.\d+)?\b`, SynNumber),
			rule(`\b(true|false|null|yes|no|on|off)\b`, SynKeyword),
			rule(`^\s*[\w.-]+\s*:`, SynType), // keys
		}}
	}

	// ── Shell ─────────────────────────────────────────────────────────────────
	for _, ext := range []string{".sh", ".bash", ".zsh"} {
		syntaxDefs[ext] = &synDef{rules: []synRule{
			rule(`#.*`, SynComment),
			rule(`"(?:[^"\\]|\\.)*"`, SynString),
			rule(`'[^']*'`, SynString),
			rule(`\b\d+\b`, SynNumber),
			rule(`\b(if|then|else|elif|fi|for|while|do|done|case|esac|in|function|return|local|export|source|alias|unset|shift|exit|break|continue)\b`, SynKeyword),
			rule(`\b(echo|printf|read|cd|ls|grep|sed|awk|find|cat|mkdir|rm|cp|mv|chmod|curl|wget|git|make|sudo)\b`, SynBuiltin),
		}}
	}

	// ── C / C++ ───────────────────────────────────────────────────────────────
	for _, ext := range []string{".c", ".h", ".cpp", ".hpp", ".cc", ".cxx"} {
		syntaxDefs[ext] = &synDef{rules: []synRule{
			rule(`//.*`, SynComment),
			rule(`/\*.*?\*/`, SynComment),
			rule(`"(?:[^"\\]|\\.)*"`, SynString),
			rule(`'(?:[^'\\]|\\.)*'`, SynString),
			rule(`\b(0x[0-9a-fA-F]+|\d+(?:\.\d+)?(?:[eE][+-]?\d+)?[uUlLfF]*)\b`, SynNumber),
			rule(`\b(auto|break|case|class|const|constexpr|continue|default|delete|do|else|enum|explicit|extern|for|friend|goto|if|inline|namespace|new|operator|private|protected|public|return|sizeof|static|struct|switch|template|this|throw|try|typedef|typename|union|using|virtual|volatile|while)\b`, SynKeyword),
			rule(`\b(bool|char|double|float|int|long|short|signed|size_t|uint8_t|uint16_t|uint32_t|uint64_t|int8_t|int16_t|int32_t|int64_t|unsigned|void|wchar_t|string|vector|map|set|pair)\b`, SynType),
			rule(`\b(NULL|nullptr|true|false|stdin|stdout|stderr|EOF)\b`, SynBuiltin),
		}}
	}

	// ── Rust ─────────────────────────────────────────────────────────────────
	syntaxDefs[".rs"] = &synDef{rules: []synRule{
		rule(`//.*`, SynComment),
		rule(`/\*.*?\*/`, SynComment),
		rule(`"(?:[^"\\]|\\.)*"`, SynString),
		rule(`'(?:[^'\\]|\\.)*'`, SynString),
		rule(`\b(0x[0-9a-fA-F_]+|0b[01_]+|\d[\d_]*(?:\.[\d_]+)?)\b`, SynNumber),
		rule(`\b(as|async|await|break|const|continue|crate|dyn|else|enum|extern|fn|for|if|impl|in|let|loop|match|mod|move|mut|pub|ref|return|self|Self|static|struct|super|trait|type|unsafe|use|where|while)\b`, SynKeyword),
		rule(`\b(bool|char|f32|f64|i8|i16|i32|i64|i128|isize|str|u8|u16|u32|u64|u128|usize|String|Vec|Option|Result|Box|Rc|Arc)\b`, SynType),
		rule(`\b(println|print|eprintln|panic|assert|assert_eq|dbg|todo|unimplemented|vec|Some|None|Ok|Err|true|false)\b`, SynBuiltin),
	}}

	// ── Markdown ──────────────────────────────────────────────────────────────
	for _, ext := range []string{".md", ".markdown"} {
		syntaxDefs[ext] = &synDef{rules: []synRule{
			rule(bt+`[^`+bt+`]*`+bt, SynString),    // inline code
			rule(`^#{1,6}\s.*`, SynKeyword),         // headings
			rule(`^\s*[-*+]\s`, SynType),            // list markers
			rule(`\*\*[^*]+\*\*`, SynBuiltin),       // bold
			rule(`\[[^\]]*\]\([^)]*\)`, SynComment), // links
		}}
	}

	// ── HTML ─────────────────────────────────────────────────────────────────
	for _, ext := range []string{".html", ".htm"} {
		syntaxDefs[ext] = &synDef{rules: []synRule{
			rule(`<!--.*?-->`, SynComment),
			rule(`"(?:[^"\\]|\\.)*"`, SynString),
			rule(`'[^']*'`, SynString),
			rule(`</?\s*[a-zA-Z][a-zA-Z0-9]*`, SynKeyword),
			rule(`\b[a-zA-Z-]+=`, SynType),
		}}
	}

	// ── CSS / SCSS ────────────────────────────────────────────────────────────
	for _, ext := range []string{".css", ".scss", ".sass", ".less"} {
		syntaxDefs[ext] = &synDef{rules: []synRule{
			rule(`/\*.*?\*/`, SynComment),
			rule(`//.*`, SynComment),
			rule(`"(?:[^"\\]|\\.)*"`, SynString),
			rule(`'[^']*'`, SynString),
			rule(`#[0-9a-fA-F]{3,8}\b`, SynNumber),
			rule(`\b\d+(?:\.\d+)?(?:px|em|rem|vh|vw|%|pt|s|ms)?\b`, SynNumber),
			rule(`\b(important|inherit|initial|unset|none|auto|normal|bold|italic)\b`, SynKeyword),
			rule(`[a-zA-Z-]+\s*:`, SynBuiltin), // property names
		}}
	}

	// ── TOML ─────────────────────────────────────────────────────────────────
	syntaxDefs[".toml"] = &synDef{rules: []synRule{
		rule(`#.*`, SynComment),
		rule(`""".*?"""`, SynString),
		rule(`"(?:[^"\\]|\\.)*"`, SynString),
		rule(`'[^']*'`, SynString),
		rule(`\b\d+(?:\.\d+)?(?:[eE][+-]?\d+)?\b`, SynNumber),
		rule(`\b(true|false)\b`, SynKeyword),
		rule(`^\[.*\]`, SynType),                          // sections
		rule(`^[a-zA-Z_][a-zA-Z0-9_.]*\s*=`, SynBuiltin), // keys
	}}

	// ── XML ───────────────────────────────────────────────────────────────────
	for _, ext := range []string{".xml", ".svg", ".xsl", ".xslt", ".xsd", ".rss", ".atom"} {
		syntaxDefs[ext] = &synDef{rules: []synRule{
			rule(`<!--.*?-->`, SynComment),
			rule(`<!\[CDATA\[.*?\]\]>`, SynString),
			rule(`"(?:[^"\\]|\\.)*"`, SynString),
			rule(`'[^']*'`, SynString),
			rule(`</?\s*[a-zA-Z][a-zA-Z0-9_:.-]*`, SynKeyword), // tags
			rule(`\b[a-zA-Z][a-zA-Z0-9_:.-]*=`, SynType),       // attributes
			rule(`&[a-zA-Z][a-zA-Z0-9]*;|&#\d+;|&#x[0-9a-fA-F]+;`, SynBuiltin), // entities
		}}
	}

	// ── PHP ───────────────────────────────────────────────────────────────────
	syntaxDefs[".php"] = &synDef{rules: []synRule{
		rule(`//.*`, SynComment),
		rule(`#.*`, SynComment),
		rule(`/\*.*?\*/`, SynComment),
		rule(`"(?:[^"\\]|\\.)*"`, SynString),
		rule(`'(?:[^'\\]|\\.)*'`, SynString),
		rule(`\b(0x[0-9a-fA-F]+|\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)\b`, SynNumber),
		rule(`\b(abstract|and|as|break|case|catch|class|clone|const|continue|declare|default|do|echo|else|elseif|enddeclare|endfor|endforeach|endif|endswitch|endwhile|extends|final|finally|for|foreach|function|global|goto|if|implements|include|include_once|instanceof|insteadof|interface|match|namespace|new|or|print|private|protected|public|readonly|require|require_once|return|static|switch|throw|trait|try|use|while|xor|yield)\b`, SynKeyword),
		rule(`\b(array|bool|boolean|callable|float|int|integer|iterable|mixed|never|null|object|resource|self|string|void)\b`, SynType),
		rule(`\b(abs|array_map|array_filter|array_keys|array_merge|array_pop|array_push|array_values|count|die|empty|exit|explode|htmlspecialchars|implode|in_array|is_array|is_null|isset|json_decode|json_encode|ltrim|nl2br|number_format|preg_match|preg_replace|rtrim|sprintf|str_contains|str_replace|strlen|strpos|strtolower|strtoupper|substr|trim|unset|var_dump|true|false|null|NULL|TRUE|FALSE)\b`, SynBuiltin),
		rule(`\$[a-zA-Z_][a-zA-Z0-9_]*`, SynBuiltin), // variables
	}}

	// ── Ruby ──────────────────────────────────────────────────────────────────
	for _, ext := range []string{".rb", ".rake", ".gemspec"} {
		syntaxDefs[ext] = &synDef{rules: []synRule{
			rule(`#.*`, SynComment),
			rule(`=begin[\s\S]*?=end`, SynComment), // multi-line comment (best-effort)
			rule(`"(?:[^"\\]|\\.)*"`, SynString),
			rule(`'(?:[^'\\]|\\.)*'`, SynString),
			rule(`:[a-zA-Z_][a-zA-Z0-9_]*`, SynString), // symbols
			rule(`\b(0x[0-9a-fA-F]+|0b[01]+|\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)\b`, SynNumber),
			rule(`\b(BEGIN|END|alias|and|begin|break|case|class|def|defined\?|do|else|elsif|end|ensure|for|if|in|module|next|not|or|redo|rescue|retry|return|self|super|then|undef|unless|until|when|while|yield)\b`, SynKeyword),
			rule(`\b(Array|Comparable|Complex|Dir|Encoding|Enumerable|Exception|File|Float|Hash|IO|Integer|Kernel|Math|Method|Module|Numeric|Object|Proc|Range|Rational|Regexp|String|Struct|Symbol|Thread|Time)\b`, SynType),
			rule(`\b(abort|at_exit|attr_accessor|attr_reader|attr_writer|block_given\?|caller|catch|exit|format|gets|lambda|loop|open|p|pp|print|printf|proc|puts|raise|rand|require|require_relative|sleep|srand|system|throw|trap|warn|nil|true|false)\b`, SynBuiltin),
		}}
	}

	// ── Makefile ──────────────────────────────────────────────────────────────
	syntaxDefs[".mk"] = &synDef{rules: []synRule{
		rule(`#.*`, SynComment),
		rule(`"(?:[^"\\]|\\.)*"`, SynString),
		rule(`\$\([^)]*\)`, SynBuiltin),                                                  // $(VAR)
		rule(`\b(ifeq|ifneq|ifdef|ifndef|else|endif|include|define|endef|export)\b`, SynKeyword),
		rule(`^[a-zA-Z_][a-zA-Z0-9_.-]*\s*[:?+]?=`, SynType),   // variable definitions
		rule(`^[a-zA-Z_][a-zA-Z0-9_.-]*\s*:`, SynKeyword),       // targets
	}}
}
