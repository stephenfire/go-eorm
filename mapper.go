package eorm

import "reflect"

type (
	ColumnMapper struct {
		fieldIndex int            // direct field index
		fieldName  string         // field of struct
		titlePath  TitlePath      // eorm tag 的值，以'/'分割
		Setter     reflect.Method // 对应的 Set 方法
		HasSetter  bool           // 是否存在对应的 Set 方法
	}

	RowMapper struct {
		// fieldIndex -> *ColumnMapper
		fields map[int]*ColumnMapper
		// fieldIndex -> mapping column indexes
		columns map[int][]int
	}
)
