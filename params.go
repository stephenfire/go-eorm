package eorm

type (
	Params struct {
		TrimSpace              bool // 是否删除首尾空格，缺省不删除
		IgnoreOutOfRange       bool // 越界时认为是零值，而不报错
		IgnoreReadRowError     bool // 迭代sheet的row时，如果出错是否跳过
		IgnoreParseError       bool // 遇到 ErrParseError 时当作无值处理
		GenWildcardForFirstRow bool // 生成 TitlePath 时，通配第一行title
		GenLastRowNoMerged     bool // 生成TitlePath时，最后一行的空不认为是横向合并
		TitleStartRow          int  // 从哪一行(行号从0开始)开始分析title_path, 所有小于0的值均被认为是0
	}

	Option func(p *Params)
)

func NewParams(opts ...Option) *Params {
	params := new(Params)
	for _, opt := range opts {
		opt(params)
	}
	return params
}

func WithTrimSpace() Option            { return func(p *Params) { p.TrimSpace = true } }
func WithIgnoreOutOfRange() Option     { return func(p *Params) { p.IgnoreOutOfRange = true } }
func WithIgnoreParseError() Option     { return func(p *Params) { p.IgnoreParseError = true } }
func WithIgnoreReadRowError() Option   { return func(p *Params) { p.IgnoreReadRowError = true } }
func WithFirstRowWildcard() Option     { return func(p *Params) { p.GenWildcardForFirstRow = true } }
func WithGenLastLayerNoMerged() Option { return func(p *Params) { p.GenLastRowNoMerged = true } }
func WithTitleStartRow(r int) Option   { return func(p *Params) { p.TitleStartRow = max(r, 0) } }

func (p *Params) MinRows(titleDepth int) int { return p.TitleStartRow + titleDepth }
