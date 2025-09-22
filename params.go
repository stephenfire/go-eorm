package eorm

type (
	Params struct {
		TrimSpace                bool // 是否删除首尾空格，缺省不删除
		IgnoreOutOfRange         bool // 越界时认为是零值，而不报错
		IgnoreReadRowError       bool // 迭代sheet的row时，如果出错是否跳过
		IgnoreParseError         bool // 遇到 ErrParseError 时当作无值处理
		GenWildcardForFirstLayer bool // 生成 TitlePath 时，通配第一行title
		GenLastLayerNoMerged     bool // 生成TitlePath时，最后一行的空不认为是横向合并
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

func WithTrimSpace() Option {
	return func(p *Params) { p.TrimSpace = true }
}

func WithIgnoreOutOfRange() Option {
	return func(p *Params) { p.IgnoreOutOfRange = true }
}

func WithIgnoreParseError() Option {
	return func(p *Params) { p.IgnoreParseError = true }
}

func WithIgnoreReadRowError() Option {
	return func(p *Params) { p.IgnoreReadRowError = true }
}

func WithGenWildcardForFirstLayer() Option {
	return func(p *Params) { p.GenWildcardForFirstLayer = true }
}

func WithGenLastLayerNoMerged() Option {
	return func(p *Params) { p.GenLastLayerNoMerged = true }
}
