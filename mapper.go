package eorm

import (
	"fmt"
	"reflect"
	"strings"
)

type (
	ColumnMapper struct {
		fieldIndex int            // direct field index
		fieldName  string         // field of struct
		titlePath  TitlePath      // eorm tag 的值，以'/'分割
		Setter     reflect.Method // 对应的 Set 方法
		HasSetter  bool           // 是否存在对应的 Set 方法
		IsSlice    bool           // 是否为数组映射
	}

	RowMapper struct {
		// fieldIndex -> *ColumnMapper
		fields map[int]*ColumnMapper
		// fieldIndex -> mapping column indexes
		columns map[int][]int
	}
)

func (m *ColumnMapper) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("[%d]", m.fieldIndex))
	sb.WriteString(m.fieldName)
	sb.WriteString(":[")
	sb.WriteString(m.titlePath.String())
	sb.WriteString("]:")
	sb.WriteString(fmt.Sprintf("HasSetter=%t", m.HasSetter))
	if m.HasSetter {
		sb.WriteString(fmt.Sprintf(",IsSlice=%t", m.IsSlice))
	}
	return sb.String()
}
