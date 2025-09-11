package eorm

type (
	Params struct {
		TrimSpace bool // 是否删除首尾空格，缺省不删除
	}

	Option func(p *Params)
)
