package eorm

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type (
	MappingType byte

	ColumnMapper struct {
		fieldIndex  int            // direct field index
		mappingType MappingType    // how to map value
		fieldType   reflect.Type   // type of the field or the setter parameter type if HasSetter is true
		fieldName   string         // field of struct
		titlePath   TitlePath      // eorm tag 的值，以'/'分割
		Setter      reflect.Method // 对应的 Set 方法
		HasSetter   bool           // 是否存在对应的 Set 方法
	}

	RowMapper[T any] struct {
		typ    reflect.Type
		params *Params
		// fieldIndex -> *ColumnMapper
		fields map[int]*ColumnMapper
		// fieldIndex -> mapping column indexes
		columns map[int][]int
	}
)

const (
	MTString MappingType = iota
	MTInt64
	MTFloat64
	MTBool
	MTStringSlice
	MTInt64Slice
	MTFloat64Slice
	MTBoolSlice
	MTInvalid
)

func NewMappingType(typ reflect.Type) (MappingType, error) {
	switch typ.Kind() {
	case reflect.Slice:
		elemType := typ.Elem()
		switch elemType.Kind() {
		case reflect.Int64:
			return MTInt64Slice, nil
		case reflect.Float64:
			return MTFloat64Slice, nil
		case reflect.String:
			return MTStringSlice, nil
		case reflect.Bool:
			return MTBoolSlice, nil
		default:
			return MTInvalid, fmt.Errorf("eorm: unsupported mapping type %s", typ.String())
		}
	case reflect.Int64:
		return MTInt64, nil
	case reflect.Float64:
		return MTFloat64, nil
	case reflect.String:
		return MTString, nil
	case reflect.Bool:
		return MTBool, nil
	default:
		return MTInvalid, fmt.Errorf("eorm: unsupported mapping type %s", typ.String())
	}
}

func (mt MappingType) IsSlice() bool {
	return mt == MTStringSlice || mt == MTInt64Slice || mt == MTFloat64Slice || mt == MTBoolSlice
}

func (mt MappingType) IsSingle() bool {
	return mt == MTString || mt == MTInt64 || mt == MTFloat64 || mt == MTBool
}

func (mt MappingType) IsValid() bool {
	return mt.IsSingle() || mt.IsSlice()
}

func (mt MappingType) String() string {
	switch mt {
	case MTStringSlice:
		return "~[]string"
	case MTInt64Slice:
		return "~[]int64"
	case MTFloat64Slice:
		return "~[]float64"
	case MTBoolSlice:
		return "~[]bool"
	case MTString:
		return "~string"
	case MTInt64:
		return "~int64"
	case MTFloat64:
		return "~float64"
	case MTBool:
		return "~bool"
	default:
		return fmt.Sprintf("N/A(0x%x)", byte(mt))
	}
}

func (m *ColumnMapper) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("[%d]", m.fieldIndex))
	sb.WriteString(m.fieldName)
	sb.WriteString(fmt.Sprintf("(%s)", m.mappingType))
	sb.WriteString(":[")
	sb.WriteString(m.titlePath.String())
	sb.WriteString("]:")
	sb.WriteString(fmt.Sprintf("HasSetter=%t", m.HasSetter))
	return sb.String()
}

func (m *ColumnMapper) SetValue(rowData reflect.Value, row Row, columnIndexes []int, params *Params) error {
	if !m.mappingType.IsValid() {
		return fmt.Errorf("eorm: invalid mapping type of column mapper %s", m.String())
	}
	if row == nil {
		return fmt.Errorf("eorm: row is nil")
	}
	if len(columnIndexes) == 0 {
		return nil
	}
	return nil
}

// findSetterMethod 查找对应的setter方法
func findSetterMethod(objType reflect.Type, fieldName string) (method reflect.Method, mtType MappingType, paramType reflect.Type, found bool) {
	setterName := "Set" + fieldName
	// 检查方法是否存在 - 首先检查指针类型的方法
	ptrType := reflect.PointerTo(objType)
	if method, ok := ptrType.MethodByName(setterName); ok {
		// 检查方法签名: func (*T) SetFieldName(string|int64|float64|bool) 或 func (*T) SetFieldName([]string|[]int64|[]float64|[]bool)
		if method.Type.NumIn() == 2 { // 接收器 + 1个参数
			paramType := method.Type.In(1)
			mtType, err := NewMappingType(paramType)
			if err == nil {
				return method, mtType, paramType, true
			}
		}
	}

	return reflect.Method{}, MTInvalid, nil, false
}

func NewRowMapper[T any](objType reflect.Type, sheet Sheet) (*RowMapper[T], *PathTree[int], error) {
	if objType.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("eorm: objType must be a struct, got %s", objType.Kind())
	}

	fieldsMapper := make(map[int]*ColumnMapper)
	pTree := new(PathTree[int])
	numFields := objType.NumField()
	for i := 0; i < numFields; i++ {
		field := objType.Field(i)

		// 检查eorm标签
		eormTag, hasEormTag := field.Tag.Lookup("eorm")
		if !hasEormTag {
			continue
		}

		// 解析title path
		titlePath, err := TitlePath(nil).Decode(eormTag)
		if err != nil {
			return nil, nil, fmt.Errorf("eorm: failed to decode title path for field %s: %w", field.Name, err)
		}
		if len(titlePath) == 0 {
			return nil, nil, fmt.Errorf("eorm: invalid title path of field %s", field.Name)
		}

		// 检查setter方法
		setterMethod, mtType, paramType, hasSetter := findSetterMethod(objType, field.Name)

		if !hasSetter {
			mtType, err = NewMappingType(field.Type)
			if err != nil {
				return nil, nil, err
			}
		} else {
			if paramType == nil {
				return nil, nil, fmt.Errorf("eorm: invalid param type for field %s setter", field.Name)
			}
			if !mtType.IsValid() {
				return nil, nil, fmt.Errorf("eorm: unsupported mapping type of filed %s", field.Name)
			}
		}

		columnMapper := &ColumnMapper{
			fieldIndex:  i,
			mappingType: mtType,
			fieldName:   field.Name,
			titlePath:   titlePath,
			Setter:      setterMethod,
			HasSetter:   hasSetter,
		}
		if hasSetter {
			columnMapper.fieldType = paramType
		} else {
			columnMapper.fieldType = field.Type
		}

		fieldsMapper[i] = columnMapper
		if err = pTree.Put(i, titlePath); err != nil {
			return nil, nil, err
		}
	}

	// 构建 fieldIndex -> []columnIndex 的映射
	// 1. 先从PathTree获取 columnIndex -> fieldIndex
	columnToField, err := MatchTitlePath(pTree, sheet)
	if err != nil {
		return nil, nil, err
	}
	// 2. 反转映射
	fieldToColumns := make(map[int][]int)
	for columnIndex, fieldIndex := range columnToField {
		fieldToColumns[fieldIndex] = append(fieldToColumns[fieldIndex], columnIndex)
	}
	// 3. 检查与 fieldsMapper 是否匹配
	for fieldIndex, columnIndexes := range fieldToColumns {
		columnMapper := fieldsMapper[fieldIndex]
		if columnMapper == nil || columnMapper.fieldIndex != fieldIndex {
			return nil, nil, fmt.Errorf("eorm: no column mapper found for field index %d", fieldIndex)
		}
		if len(columnIndexes) > 1 {
			if !columnMapper.mappingType.IsSlice() {
				return nil, nil, fmt.Errorf("eorm: a slice mapping type is needed for multi-columns at field index %d", fieldIndex)
			}
		}

		sort.Ints(columnIndexes)
	}

	return &RowMapper[T]{
		typ:     objType,
		fields:  fieldsMapper,
		columns: fieldToColumns,
	}, pTree, nil
}

func (m *RowMapper[T]) Transit(row Row) (*T, error) {
	if row == nil {
		return nil, nil
	}
	val := reflect.New(m.typ)

	for fieldIndex, columnIndexes := range m.columns {
		if len(columnIndexes) == 0 {
			continue
		}
		columnMapper := m.fields[fieldIndex]
		if columnMapper == nil {
			return nil, fmt.Errorf("eorm: no column mapper found for field index %d", fieldIndex)
		}
		if err := columnMapper.SetValue(val, row, columnIndexes, m.params); err != nil {
			return nil, err
		}
	}
	return val.Interface().(*T), nil
}
