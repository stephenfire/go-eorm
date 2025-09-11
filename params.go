package eorm

type (
	Params struct {
		TrimSpace    bool // 是否删除首尾空格，缺省不删除
		NoOutOfRange bool // 越界时设为0值，而不报错
	}

	Option func(p *Params)
)
