package eorm

import (
	"fmt"
	"reflect"
)

type EORMWriter[T any] struct {
	w           SheetWriter
	rowMapper   *RowMapper[T]
	curRowIndex int
}

func NewWriter[T any](w SheetWriter, objType reflect.Type, opts ...Option) (*EORMWriter[T], error) {
	// 检查objType是否为结构体
	if objType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("eorm: objType must be a struct, got %s", objType.Kind())
	}

	opts = append(opts, WithWrite())
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
		w:           w,
		rowMapper:   rowMapper,
		curRowIndex: stream.CurrentRowNumber(),
	}, nil
}

// Append write all not nil objects from curRowIndex in input order. returns the number of wrote objects.
func (w *EORMWriter[T]) Append(objs ...*T) (int, error) {
	n := 0
	for _, obj := range objs {
		if obj == nil {
			continue
		}
		nextRow := w.curRowIndex + 1
		if err := w.rowMapper.Write(w.w, nextRow, obj); err != nil {
			return n, err
		}
		n += 1
		w.curRowIndex += 1
	}
	return n, nil
}

func (w *EORMWriter[T]) SkipRows(n int) {
	w.curRowIndex += n
}

func (w *EORMWriter[T]) CurrentRowIndex() int {
	return w.curRowIndex
}
