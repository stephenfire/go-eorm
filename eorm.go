package eorm

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type EORM[T any] struct {
	sheet           Sheet
	objType         reflect.Type
	params          *Params
	rowMapper       *RowMapper[T]
	columnTree      *PathTree[int]
	currentIterator RowIterator
	currentRow      Row
	currentObj      *T
	sheetIndex      int
	rowIndex        int
	initialized     bool
}

func NewEORM[T any](sheet Sheet, objType reflect.Type, opts ...Option) (*EORM[T], error) {
	// 首先检查objType是否为结构体
	if objType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("eorm: objType must be a struct, got %s", objType.Kind())
	}

	// 处理选项参数
	params := &Params{}
	for _, opt := range opts {
		opt(params)
	}

	// 分析对象类型，创建ColumnMapper
	rowMapper, columnTree, err := NewRowMapper[T](objType, sheet)
	if err != nil {
		return nil, err
	}

	return &EORM[T]{
		sheet:      sheet,
		objType:    objType,
		params:     params,
		rowMapper:  rowMapper,
		columnTree: columnTree,
	}, nil
}

// Next 移动到下一行，如果还有行则返回true，否则返回false
func (e *EORM[T]) Next() bool {
	// 如果没有初始化迭代器，先初始化
	if !e.initialized {
		if err := e.initializeIterator(); err != nil {
			return false
		}
		e.initialized = true
	}

	// 如果当前迭代器已经结束，尝试下一个工作表
	for e.currentIterator != nil {
		if e.currentIterator.Next() {
			e.rowIndex++
			row, err := e.currentIterator.Current()
			if err != nil {
				// 处理错误，继续下一行
				continue
			}
			e.currentRow = row
			e.currentObj = e.mapRowToObject(row)
			return true
		}

		// 当前迭代器结束，关闭并尝试下一个工作表
		_ = e.currentIterator.Close()
		e.currentIterator = nil
		e.sheetIndex++
		if e.sheetIndex >= e.wb.SheetCount() {
			// 所有工作表都遍历完了
			return false
		}

		// 初始化下一个工作表的迭代器
		it, err := e.wb.IterateSheet(e.sheetIndex)
		if err != nil {
			// 处理错误，继续下一个工作表
			continue
		}
		e.currentIterator = it
		e.rowIndex = 0
	}

	return false
}

// Current 返回当前行的对象
func (e *EORM[T]) Current() *T {
	return e.currentObj
}

// initializeIterator 初始化行迭代器
func (e *EORM[T]) initializeIterator() error {
	if e.wb.SheetCount() == 0 {
		return fmt.Errorf("no sheets available")
	}

	it, err := e.wb.IterateSheet(e.sheetIndex)
	if err != nil {
		return err
	}
	e.currentIterator = it
	return nil
}

// mapRowToObject 将行数据映射到对象
func (e *EORM[T]) mapRowToObject(row Row) *T {
	obj := new(T)
	objValue := reflect.ValueOf(obj).Elem()

	// 遍历所有字段映射
	for fieldIndex, columnMapper := range e.rowMapper.fields {
		if fieldIndex < 0 || fieldIndex >= objValue.NumField() {
			continue
		}

		// 从预构建的列映射获取列索引
		columnIndex, exists := e.columnMapping[fieldIndex]
		if !exists {
			continue
		}

		field := objValue.Field(fieldIndex)

		// 获取列值
		columnValue, err := row.GetColumn(columnIndex)
		if err != nil {
			continue
		}

		// 根据字段类型设置值
		if columnMapper.HasSetter {
			// 使用setter方法
			e.callSetterMethod(obj, columnMapper, columnValue)
		} else {
			// 直接设置字段值
			e.setFieldValue(field, columnValue)
		}
	}

	return obj
}

// getColumnIndexForField 根据title path获取列索引
func (e *EORM[T]) getColumnIndexForField(titlePath TitlePath, row Row) (int, error) {
	// 这里需要实现根据title path找到对应列索引的逻辑
	// 暂时返回一个简单的实现：假设title path的最后一部分是列名
	if len(titlePath) == 0 {
		return 0, fmt.Errorf("empty title path")
	}

	// 简单实现：遍历所有列，查找匹配的列名
	columnName := titlePath[len(titlePath)-1]
	columnCount := row.ColumnCount()

	for i := 0; i < columnCount; i++ {
		cellValue, err := row.GetColumn(i)
		if err != nil {
			continue
		}
		if cellValue == columnName {
			return i, nil
		}
	}

	return 0, fmt.Errorf("column not found: %s", columnName)
}

// callSetterMethod 调用setter方法设置值
func (e *EORM[T]) callSetterMethod(obj *T, columnMapper *ColumnMapper, value string) {
	objValue := reflect.ValueOf(obj)
	method := columnMapper.Setter

	if columnMapper.IsSlice {
		// 数组setter方法
		values := []string{value}
		method.Func.Call([]reflect.Value{objValue, reflect.ValueOf(values)})
	} else {
		// 单值setter方法
		method.Func.Call([]reflect.Value{objValue, reflect.ValueOf(value)})
	}
}

// setFieldValue 直接设置字段值
func (e *EORM[T]) setFieldValue(field reflect.Value, value string) {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intValue, err := parseStringToInt(value); err == nil {
			field.SetInt(intValue)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if uintValue, err := parseStringToUint(value); err == nil {
			field.SetUint(uintValue)
		}
	case reflect.Float32, reflect.Float64:
		if floatValue, err := parseStringToFloat(value); err == nil {
			field.SetFloat(floatValue)
		}
	case reflect.Bool:
		if boolValue, err := parseStringToBool(value); err == nil {
			field.SetBool(boolValue)
		}
	}
}

// parseStringToInt 将字符串解析为int64
func parseStringToInt(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("eorm: parse string to int64 failed: %w", err)
	}
	return i, nil
}

// parseStringToUint 将字符串解析为uint64
func parseStringToUint(s string) (uint64, error) {
	if s == "" {
		return 0, nil
	}
	u, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("eorm: parse string to uint64 failed: %w", err)
	}
	return u, nil
}

// parseStringToFloat 将字符串解析为float64
func parseStringToFloat(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("eorm: parse string to float64 failed: %w", err)
	}
	return f, nil
}

// parseStringToBool 将字符串解析为bool
func parseStringToBool(s string) (bool, error) {
	if s == "" {
		return false, nil
	}
	s = strings.ToUpper(s)
	switch s {
	case "TRUE", "1", "YES", "Y":
		return true, nil
	case "FALSE", "0", "NO", "N":
		return false, nil
	default:
		return false, fmt.Errorf("eorm: parse string to bool failed: unknown value: %s", s)
	}
}

// singleRowSheet 包装单个行作为Sheet接口实现
type singleRowSheet struct {
	row Row
}

func (s *singleRowSheet) RowCount() int {
	return 1
}

func (s *singleRowSheet) GetRow(index int) (Row, error) {
	if index != 0 {
		return nil, ErrOutOfRange
	}
	return s.row, nil
}
