package eorm

import (
	"fmt"
	"reflect"
)

type EORMWriter[T any] struct {
	w         SheetWriter
	rowMapper *RowMapper[T]
}

func NewWriter[T any](w SheetWriter, objType reflect.Type, opts ...Option) (*EORMWriter[T], error) {
	// 检查objType是否为结构体
	if objType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("eorm: objType must be a struct, got %s", objType.Kind())
	}

	params := NewParams(opts...)
	stream, err := w.Stream()
	if err != nil {
		return nil, err
	}

	// 分析对象类型，创建ColumnMapper
	rowMapper, _, err := NewRowMapperByStream[T](objType, stream, params)
	if err != nil {
		return nil, err
	}

	return &EORMWriter[T]{
		w:         w,
		rowMapper: rowMapper,
	}, nil
}
