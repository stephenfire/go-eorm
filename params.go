package eorm

type (
	Params struct {
		TrimSpace              bool       // 是否删除首尾空格，缺省不删除
		IgnoreOutOfRange       bool       // 越界时认为是零值，而不报错
		IgnoreReadRowError     bool       // 迭代sheet的row时，如果出错是否跳过
		IgnoreParseError       bool       // 遇到 ErrParseError 时当作无值处理
		GenWildcardForFirstRow bool       // 生成 TitlePath 时，通配第一行title
		GenLastRowNoMerged     bool       // 生成TitlePath时，最后一行的空不认为是横向合并
		TitleStartRow          int        // 从哪一行(行号从0开始)开始分析title_path, 所有小于0的值均被认为是0
		RequiredMatchLevel     MatchLevel // 需要类型与excel表头的匹配程度，无论何值，tag.constraint的要求必须达成
	}

	Option func(p *Params)

	MatchLevel byte
)

const (
	MatchLevelNone    MatchLevel = iota // 无要求
	MatchLevelMatched                   // 存在匹配字段即可
	MatchLevelPerfect                   // 类型所有具有"eorm"标签的字段均已找到匹配的列
)

func NewParams(opts ...Option) *Params {
	params := new(Params)
	for _, opt := range opts {
		opt(params)
	}
	return params
}

func WithTrimSpace() Option              { return func(p *Params) { p.TrimSpace = true } }
func WithIgnoreOutOfRange() Option       { return func(p *Params) { p.IgnoreOutOfRange = true } }
func WithIgnoreParseError() Option       { return func(p *Params) { p.IgnoreParseError = true } }
func WithIgnoreReadRowError() Option     { return func(p *Params) { p.IgnoreReadRowError = true } }
func WithFirstRowWildcard() Option       { return func(p *Params) { p.GenWildcardForFirstRow = true } }
func WithGenLastLayerNoMerged() Option   { return func(p *Params) { p.GenLastRowNoMerged = true } }
func WithTitleStartRow(r int) Option     { return func(p *Params) { p.TitleStartRow = max(r, 0) } }
func WithMatchLevel(l MatchLevel) Option { return func(p *Params) { p.RequiredMatchLevel = l } }
func WithParams(src *Params) Option      { return func(p *Params) { p.CopyFrom(src) } }

func (p *Params) MinRows(titleDepth int) int { return p.TitleStartRow + titleDepth }

func (p *Params) CopyFrom(src *Params) *Params {
	p.TrimSpace = src.TrimSpace
	p.IgnoreOutOfRange = src.IgnoreOutOfRange
	p.IgnoreReadRowError = src.IgnoreReadRowError
	p.IgnoreParseError = src.IgnoreParseError
	p.GenWildcardForFirstRow = src.GenWildcardForFirstRow
	p.GenLastRowNoMerged = src.GenLastRowNoMerged
	p.TitleStartRow = src.TitleStartRow
	p.RequiredMatchLevel = src.RequiredMatchLevel
	return p
}

func (l MatchLevel) Formalize() MatchLevel {
	if l <= MatchLevelNone {
		return MatchLevelNone
	}
	if l == MatchLevelMatched {
		return MatchLevelMatched
	}
	return MatchLevelPerfect
}
