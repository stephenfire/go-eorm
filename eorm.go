package eorm

import (
	"errors"
	"fmt"
	"iter"
	"reflect"

	"github.com/stephenfire/go-common"
)

const (
	Version   = common.Version(1000)
	Copyright = "Copyright 2025 stephen.fire@gmail.com"
)

var (
	ErrEmptyPath              = errors.New("eorm: empty path")
	ErrUnsupported            = errors.New("eorm: unsupported")
	ErrInvalidState           = errors.New("eorm: invalid state")
	ErrRowNotFound            = errors.New("eorm: row not found")
	ErrRequiredColumnNotFound = errors.New("eorm: required column not found")
)

type EORM[T any] struct {
	sheet      Sheet
	objType    reflect.Type
	params     *Params
	rowMapper  *RowMapper[T]
	columnTree *PathTree[int]
	currentRow Row
	currentObj *T
	rowIndex   int
	lastErr    error
}

func NewEORM[T any](sheet Sheet, objType reflect.Type, opts ...Option) (*EORM[T], error) {
	// 检查objType是否为结构体
	if objType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("eorm: objType must be a struct, got %s", objType.Kind())
	}

	params := NewParams(opts...)

	// 分析对象类型，创建ColumnMapper
	rowMapper, columnTree, err := NewRowMapper[T](objType, sheet, params)
	if err != nil {
		return nil, err
	}

	return &EORM[T]{
		sheet:      sheet,
		objType:    objType,
		params:     params,
		rowMapper:  rowMapper,
		columnTree: columnTree,
		rowIndex:   -1,
	}, nil
}

func (e *EORM[T]) IsValid() bool {
	if e == nil || e.sheet == nil || e.objType == nil || e.rowMapper == nil || e.columnTree == nil {
		return false
	}
	if e.columnTree.Depth() < 1 {
		return false
	}
	return true
}

func (e *EORM[T]) IsPerfectMatch() bool {
	if e == nil || e.rowMapper == nil {
		return false
	}
	return e.rowMapper.IsPerfectMatch()
}

func (e *EORM[T]) IsMatched() bool {
	if e == nil || e.rowMapper == nil {
		return false
	}
	return e.rowMapper.IsMatched()
}

func (e *EORM[T]) LastError() error  { return e.lastErr }
func (e *EORM[T]) ClrLastError()     { e.lastErr = nil }
func (e *EORM[T]) DataStartRow() int { return e.params.MinRows(e.columnTree.Depth()) }

// Next 移动到下一行，如果还有行则返回true，否则返回false
func (e *EORM[T]) Next() bool {
	if !e.IsValid() {
		return false
	}
	// 如果没有初始化迭代器，先初始化
	if e.rowIndex == -1 {
		startRow := e.DataStartRow()
		if startRow >= e.sheet.RowCount() {
			return false
		}
		// 因为遍历时先自增，所以这里-1。又因为tree depth不可能小于1，所以这个值不会小于0
		e.rowIndex = startRow - 1
	}
	if e.rowIndex < 0 || e.rowIndex >= e.sheet.RowCount() {
		return false
	}

	e.currentRow = nil
	e.currentObj = nil
	e.lastErr = nil
	for e.rowIndex >= 0 && e.rowIndex < e.sheet.RowCount() {
		e.rowIndex++
		if e.rowIndex >= e.sheet.RowCount() {
			e.rowIndex = -2
			return false
		}
		row, err := e.sheet.GetRow(e.rowIndex)
		if err != nil {
			e.lastErr = err
			if e.params.IgnoreReadRowError {
				continue
			} else {
				return false
			}
		}
		if row == nil {
			continue
		}
		e.currentRow = row
		return true
	}

	return false
}

func (e *EORM[T]) CheckValue() error {
	if !e.IsValid() || e.rowIndex < 0 || e.rowIndex >= e.sheet.RowCount() {
		return ErrInvalidState
	}
	if e.currentRow == nil {
		return ErrRowNotFound
	}
	return nil
}

// Current 返回当前行的对象
func (e *EORM[T]) Current() (*T, error) {
	if err := e.CheckValue(); err != nil {
		return nil, err
	}
	if e.currentObj != nil {
		return e.currentObj, nil
	}
	obj, err := e.rowMapper.Transit(e.currentRow)
	if err != nil {
		e.lastErr = err
		return nil, err
	}
	e.currentObj = obj
	return e.currentObj, nil
}

func (e *EORM[T]) All() iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		for e.Next() {
			t, err := e.Current()
			if !yield(t, err) {
				return
			}
		}
	}
}
