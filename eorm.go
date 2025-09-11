package eorm

import (
	"fmt"
	"reflect"
)

type EORM[T any] struct {
	wb         Workbook
	objType    reflect.Type
	params     *Params
	rowMapper  *RowMapper
	columnTree *PathTree[int]
}

func (e *EORM[T]) Close() error {
	if e != nil && e.wb != nil {
		return e.wb.Close()
	}
	return nil
}

func NewEORM[T any](filePath string, objType reflect.Type, opts ...Option) (*EORM[T], error) {
	// 首先检查objType是否为结构体
	if objType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("eorm: objType must be a struct, got %s", objType.Kind())
	}

	// 处理选项参数
	params := &Params{}
	for _, opt := range opts {
		opt(params)
	}

	wb, err := NewWorkbook(filePath)
	if err != nil {
		return nil, err
	}

	// 分析对象类型，创建ColumnMapper
	rowMapper, err := analyzeObjectType(objType)
	if err != nil {
		_ = wb.Close()
		return nil, err
	}

	// 创建列映射树
	columnTree, err := buildColumnTree(rowMapper)
	if err != nil {
		_ = wb.Close()
		return nil, fmt.Errorf("failed to build column tree: %w", err)
	}

	return &EORM[T]{
		wb:         wb,
		objType:    objType,
		params:     params,
		rowMapper:  rowMapper,
		columnTree: columnTree,
	}, nil
}

// analyzeObjectType 分析对象类型，提取eorm标签信息
func analyzeObjectType(objType reflect.Type) (*RowMapper, error) {
	if objType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("eorm: objType must be a struct, got %s", objType.Kind())
	}

	rowMapper := &RowMapper{
		fields:  make(map[int]*ColumnMapper),
		columns: make(map[int][]int),
	}

	numFields := objType.NumField()
	for i := 0; i < numFields; i++ {
		field := objType.Field(i)

		// 检查eorm标签
		eormTag, hasEormTag := field.Tag.Lookup("eorm")
		if !hasEormTag {
			continue // 跳过没有eorm标签的字段
		}

		// 解析title path
		titlePath, err := TitlePath(nil).Decode(eormTag)
		if err != nil {
			return nil, fmt.Errorf("eorm: failed to decode title path for field %s: %w", field.Name, err)
		}
		if len(titlePath) == 0 {
			return nil, fmt.Errorf("eorm: invalid title path of field %s", field.Name)
		}

		// 检查setter方法
		setterMethod, sliceParam, hasSetter := findSetterMethod(objType, field.Name)

		columnMapper := &ColumnMapper{
			fieldIndex: i,
			fieldName:  field.Name,
			titlePath:  titlePath,
			Setter:     setterMethod,
			HasSetter:  hasSetter,
			IsSlice:    sliceParam,
		}

		rowMapper.fields[i] = columnMapper
	}

	return rowMapper, nil
}

// findSetterMethod 查找对应的setter方法
func findSetterMethod(objType reflect.Type, fieldName string) (method reflect.Method, sliceParam bool, found bool) {
	// 查找单值setter方法
	setterName := "Set" + fieldName

	// 检查方法是否存在 - 首先检查指针类型的方法
	ptrType := reflect.PointerTo(objType)
	if method, ok := ptrType.MethodByName(setterName); ok {
		// 检查方法签名: func (*T) SetFieldName(string) 或 func (*T) SetFieldName([]string)
		if method.Type.NumIn() == 2 { // 接收器 + 1个参数
			paramType := method.Type.In(1)
			if paramType.Kind() == reflect.String {
				return method, false, true
			} else if paramType.Kind() == reflect.Slice && paramType.Elem().Kind() == reflect.String {
				return method, true, true
			}
		}
	}

	return reflect.Method{}, false, false
}

// buildColumnTree 构建列映射树
func buildColumnTree(rowMapper *RowMapper) (*PathTree[int], error) {
	tree := &PathTree[int]{}
	for fieldIndex, columnMapper := range rowMapper.fields {
		err := tree.Put(fieldIndex, columnMapper.titlePath)
		if err != nil {
			return nil, err
		}
	}
	return tree, nil
}
