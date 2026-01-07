package eorm

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/stephenfire/go-tools"
)

type (
	MappingType byte
	Constraint  string

	// callingMethod 用来记录Setter或Getter的对应方法和参数或返回值的类型
	// 当Setter或Getter方法不存在时，_type和mappingType则为属性值类型和映射类型
	callingMethod struct {
		_type       reflect.Type   // 参数或返回值的类型
		mappingType MappingType    // 参数或返回值的映射类型
		method      reflect.Method // 对应方法
		exist       bool           // 对应方法是否存在
	}

	// ColumnMapper 记录对象属性与表格列之间的映射关系
	ColumnMapper struct {
		fieldIndex int           // direct field index
		fieldName  string        // field of struct
		titlePath  TitlePath     // eorm tag 的值，以'/'分割
		constraint Constraint    // "" or required or not_null
		setter     callingMethod // 向对象对应属性中写入值时需要的反射信息
		getter     callingMethod // 从对象中读取对应属性值时需要的反射信息
	}

	// RowMapper RowMapper[T]对象的主要功能是把Row转换为一个类型为*T的对象。其中：
	//
	// * RowMapper.typ属性是T类型的reflect.Type。
	// * RowMapper.fields保存所有类型T中所有需要映射的属性和信息ColumnMapper
	// * RowMapper.columns保存T中每一个需要映射的属性值需要由Row中哪些列的值构成。
	//
	// ColumnMapper中保存了属性值的构成方法，分为两种：
	//
	// * 当ColumnMapper.HasSetter==false时，直接赋值给属性值
	// * 当ColumnMapper.HasSetter==true时，通过ColumnMapper.Setter保存的*T的方法设置属性值。
	//
	// 无论是哪种方式，值的类型都保存在ColumnMapper.fieldType中，而值类型的Kind()只能是string, int64, float64, bool, []string, []
	// int64, []float64, []bool之一。
	//
	// 转换方法为RowMapper.Transit(row Row) (*T, error)方法，其步骤为：
	//
	// 1. 遍历由对象属性fieldIndex到Row中列columnIndexes的映射表RowMapper.columns，当映射到多列时，也就是len(columnIndexes)>
	//   1，值类型必须是[]string, []int64, []float64, []bool之一。
	// 2. 遍历columnIndexes，从row中获取各列对应的值，并转换为ColumnMapper.fieldType的类型，得到fieldValue
	// 3. 创建RowMapper.typ类型对应的指针对象rowData
	// 4. 当ColumnMapper.HasSetter==false时，将fieldValue直接赋值给rowData对应index为fieldIndex的属性
	// 5. 当ColumnMapper.HasSetter==true时，将fieldValue传递给rowData对象对应的ColumnMapper.Setter方法，完成值设置。
	// 6. 返回新创建的rowData
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

const (
	ConstraintDefault  = ""
	ConstraintRequired = "required"
	ConstraintNotNull  = "not_null"
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

func (c Constraint) IsValid() bool {
	return c == ConstraintDefault || c == ConstraintRequired || c == ConstraintNotNull
}

func (c Constraint) NeedMapper() bool {
	return c == ConstraintRequired || c == ConstraintNotNull
}

func (c Constraint) NeedValue() bool {
	return c == ConstraintNotNull
}

func (c Constraint) String() string {
	return string(c)
}

func newCallingMethod(field reflect.StructField, isSetter bool,
	exist bool, method reflect.Method, mtType MappingType, valType reflect.Type) (callingMethod, error) {
	if !exist {
		fMtType, err := NewMappingType(field.Type)
		if err != nil {
			return callingMethod{}, err
		}
		return callingMethod{
			_type:       field.Type,
			mappingType: fMtType,
			method:      reflect.Method{},
			exist:       false,
		}, nil
	} else {
		if valType == nil {
			return callingMethod{}, fmt.Errorf("eorm: invalid param type for field %s %s",
				field.Name, tools.IF(isSetter, "setter", "getter"))
		}
		if !mtType.IsValid() {
			return callingMethod{}, fmt.Errorf("eorm: unsupported %s mapping type of field %s",
				tools.IF(isSetter, "setter", "getter"), field.Name)
		}
		return callingMethod{
			_type:       valType,
			mappingType: mtType,
			method:      method,
			exist:       true,
		}, nil
	}
}

func (c callingMethod) isValid() bool {
	return c._type != nil
}

func NewColumnMapper(objType reflect.Type, fieldIdx int, field reflect.StructField, params *Params) (*ColumnMapper, error) {
	titlePath, constraint, err := titlePathInTag(field)
	if err != nil {
		return nil, err
	}
	if len(titlePath) == 0 {
		return nil, nil
	}

	var setter, getter callingMethod

	// 检查setter方法
	setterMethod, mtType, paramType, hasSetter := findSetterMethod(objType, field.Name)
	setter, err = newCallingMethod(field, true, hasSetter, setterMethod, mtType, paramType)
	if err != nil {
		return nil, err
	}

	if params != nil && params.Writable {
		// getter method
		getterMethod, getterMtType, returnType, hasGetter := findGetterMethod(objType, field.Name)
		getter, err = newCallingMethod(field, false, hasGetter, getterMethod, getterMtType, returnType)
		if err != nil {
			return nil, err
		}
	}

	columnMapper := &ColumnMapper{
		fieldIndex: fieldIdx,
		fieldName:  field.Name,
		titlePath:  titlePath,
		constraint: Constraint(constraint),
		setter:     setter,
		getter:     getter,
	}

	return columnMapper, nil
}

func (m *ColumnMapper) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("[%d]", m.fieldIndex))
	sb.WriteString(m.fieldName)
	sb.WriteString(":[")
	sb.WriteString(m.titlePath.String())
	sb.WriteString("]")
	if m.setter.isValid() {
		sb.WriteString(fmt.Sprintf(", Setter(%s)", m.setter.mappingType.String()))
		if m.setter.exist {
			sb.WriteString(fmt.Sprintf("(%s)", m.setter.method.Name))
		}
	}
	if m.getter.isValid() {
		sb.WriteString(fmt.Sprintf(", Getter(%s)", m.getter.mappingType.String()))
		if m.getter.exist {
			sb.WriteString(fmt.Sprintf("(%s)", m.getter.method.Name))
		}
	}
	return sb.String()
}

// CellToField read data from row and set data to mapped object rowData
func (m *ColumnMapper) CellToField(rowData reflect.Value, row Row, columnIndexes []int, params *Params) error {
	if !m.setter.mappingType.IsValid() {
		return fmt.Errorf("eorm: invalid mapping type of column mapper %s", m.String())
	}
	if row == nil {
		return errors.New("eorm: row is nil")
	}
	if len(columnIndexes) == 0 {
		return nil
	}

	// 获取字段值
	var fieldValue reflect.Value
	var err error

	if m.setter.mappingType.IsSlice() {
		// 处理切片类型
		fieldValue, err = m.getSliceCellValue(row, columnIndexes, params)
	} else {
		// 处理单值类型
		if len(columnIndexes) > 1 {
			return fmt.Errorf("eorm: single value mapping type requires exactly one column, got %d", len(columnIndexes))
		}
		fieldValue, err = m.getSingleCellValue(row, columnIndexes[0], params)
	}

	if err != nil {
		return err
	}
	if !fieldValue.IsValid() {
		return nil
	}

	// 设置字段值
	if m.setter.exist {
		// 调用 Setter 方法
		method := m.setter.method
		methodValue := rowData.MethodByName(method.Name)
		if !methodValue.IsValid() {
			return fmt.Errorf("eorm: setter method %s not found", method.Name)
		}
		methodValue.Call([]reflect.Value{fieldValue})
	} else {
		// 直接设置字段值
		field := rowData.Elem().Field(m.fieldIndex)
		if !field.CanSet() {
			return fmt.Errorf("eorm: field %s is not settable", m.fieldName)
		}
		field.Set(fieldValue)
	}

	return nil
}

func cellToValue[T any](fn func(index int) (T, error), index int, constraint Constraint) (reflect.Value, error) {
	v, e := fn(index)
	if e != nil {
		if !constraint.NeedValue() && errors.Is(e, ErrEmptyCell) {
			return reflect.ValueOf(v), nil
		}
		return reflect.Value{}, e
	}
	val := reflect.ValueOf(v)
	if constraint.NeedValue() && (!val.IsValid() || val.IsZero()) {
		return reflect.Value{}, ErrEmptyCell
	}
	return reflect.ValueOf(v), nil
}

func (m *ColumnMapper) cellValue(getter func(Row, int) (reflect.Value, error),
	valueType reflect.Type, row Row, columnIndex int, params *Params) (reflect.Value, error) {
	val, err := getter(row, columnIndex)
	if m.constraint.NeedValue() && (err != nil || !val.IsValid() || val.IsZero()) {
		return reflect.Value{}, ErrEmptyCell
	}
	if err != nil {
		if (params.IgnoreParseError && errors.Is(err, ErrParseError)) ||
			(params.IgnoreOutOfRange && errors.Is(err, ErrOutOfRange)) {
			return reflect.Zero(valueType), nil
		}
		return reflect.Value{}, err
	}
	if !val.IsValid() {
		val = reflect.Zero(valueType)
	} else if val.Type() != valueType {
		val = val.Convert(valueType)
	}
	return val, nil
}

func (m *ColumnMapper) getSingleCellValue(row Row, columnIndex int, params *Params) (reflect.Value, error) {
	singleMap := func(getter func(row Row, index int) (reflect.Value, error)) (reflect.Value, error) {
		return m.cellValue(getter, m.setter._type, row, columnIndex, params)
	}
	switch m.setter.mappingType {
	case MTString:
		return singleMap(func(row Row, index int) (reflect.Value, error) {
			if params.TrimSpace {
				return cellToValue(func(index int) (string, error) {
					v, err := row.GetColumn(index)
					if err != nil {
						return v, err
					}
					return strings.TrimSpace(v), nil
				}, columnIndex, m.constraint)
			}
			return cellToValue(row.GetColumn, columnIndex, m.constraint)
		})
	case MTInt64:
		return singleMap(func(row Row, index int) (reflect.Value, error) {
			return cellToValue(row.GetInt64Column, columnIndex, m.constraint)
		})
	case MTFloat64:
		return singleMap(func(row Row, index int) (reflect.Value, error) {
			return cellToValue(row.GetFloat64Column, columnIndex, m.constraint)
		})
	case MTBool:
		return singleMap(func(row Row, index int) (reflect.Value, error) {
			return cellToValue(row.GetBoolColumn, columnIndex, m.constraint)
		})
	default:
		return reflect.Value{}, fmt.Errorf("eorm: unsupported single value mapping type: %s", m.setter.mappingType)
	}
}

func (m *ColumnMapper) getSliceCellValue(row Row, columnIndexes []int, params *Params) (reflect.Value, error) {
	sliceMap := func(getter func(row Row, index int) (reflect.Value, error)) (reflect.Value, error) {
		slice := reflect.MakeSlice(m.setter._type, len(columnIndexes), len(columnIndexes))
		for i, colIdx := range columnIndexes {
			val, err := m.cellValue(getter, m.setter._type.Elem(), row, colIdx, params)
			if err != nil {
				return reflect.Value{}, err
			}
			slice.Index(i).Set(val)
		}
		return slice, nil
	}

	switch m.setter.mappingType {
	case MTStringSlice:
		return sliceMap(func(row Row, index int) (reflect.Value, error) {
			if params.TrimSpace {
				return cellToValue(func(index int) (string, error) {
					v, err := row.GetColumn(index)
					if err != nil {
						return "", err
					}
					return strings.TrimSpace(v), nil
				}, index, m.constraint)
			}
			return cellToValue(row.GetColumn, index, m.constraint)
		})
	case MTInt64Slice:
		return sliceMap(func(row Row, index int) (reflect.Value, error) {
			return cellToValue(row.GetInt64Column, index, m.constraint)
		})
	case MTFloat64Slice:
		return sliceMap(func(row Row, index int) (reflect.Value, error) {
			return cellToValue(row.GetFloat64Column, index, m.constraint)
		})
	case MTBoolSlice:
		return sliceMap(func(row Row, index int) (reflect.Value, error) {
			return cellToValue(row.GetBoolColumn, index, m.constraint)
		})
	default:
		return reflect.Value{}, fmt.Errorf("eorm: unsupported slice mapping type: %s", m.setter.mappingType)
	}
}

// fieldValue 获取obj对象中对应属性的值，通过 getter 方法（如果存在），或直接返回属性值。
// 此时 obj 应是绑定getter方法的指针类型，当需要直接获取属性值时，需要去掉指针
func (m *ColumnMapper) fieldValue(obj reflect.Value) (reflect.Value, error) {
	if m.getter.exist {
		method := m.getter.method
		methodValue := obj.MethodByName(method.Name)
		if !methodValue.IsValid() {
			return reflect.Value{}, fmt.Errorf("eorm: getter method %s not found", method.Name)
		}
		return methodValue.Call([]reflect.Value{})[0], nil
	} else {
		return obj.Elem().Field(m.fieldIndex), nil
	}
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

// findGetterMethod 查找类型属性与setter方法对应的getter方法
func findGetterMethod(objType reflect.Type, fieldName string) (method reflect.Method, mtType MappingType, returnType reflect.Type, found bool) {
	getterName := "Get" + fieldName
	// 检查方法是否存在 - 首先检查指针类型的方法
	ptrType := reflect.PointerTo(objType)
	if method, ok := ptrType.MethodByName(getterName); ok {
		// 检查方法签名: func (*T) GetFieldName() string|int64|float64|bool 或 func (*T) GetFieldName() []string|[]int64|[]float64|[]bool
		if method.Type.NumIn() == 1 && method.Type.NumOut() == 1 { // 1 receiver and 1 return value
			returnType := method.Type.Out(0)
			mtType, err := NewMappingType(returnType)
			if err == nil {
				return method, mtType, returnType, true
			}
		}
	}

	return reflect.Method{}, MTInvalid, nil, false
}

func titlePathInTag(field reflect.StructField) (TitlePath, string, error) {
	// 检查eorm标签
	eormTag, hasEormTag := field.Tag.Lookup("eorm")
	if !hasEormTag {
		return nil, "", nil
	}

	titlepathTag := eormTag
	constraint := ""
	parts := strings.SplitN(eormTag, ",", 2)
	if len(parts) > 1 {
		titlepathTag = parts[0]
		constraint = parts[1]
		if !Constraint(constraint).IsValid() {
			constraint = ""
		}
	}

	// 解析title path
	titlePath, err := TitlePath(nil).Decode(titlepathTag)
	if err != nil {
		return nil, "", fmt.Errorf("eorm: failed to decode title path for field %s: %w", field.Name, err)
	}
	if len(titlePath) == 0 {
		return nil, "", fmt.Errorf("eorm: invalid title path of field %s", field.Name)
	}
	return titlePath, constraint, nil
}

func newRowMapper[T any](
	objType reflect.Type,
	titlePathMatcher func(tree *PathTree[int], params *Params) (columnToField map[int]int, err error),
	params *Params) (*RowMapper[T], *PathTree[int], error) {
	if objType.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("eorm: objType must be a struct, got %s", objType.Kind())
	}

	fieldsMapper := make(map[int]*ColumnMapper)
	pTree := new(PathTree[int])
	numFields := objType.NumField()
	for i := 0; i < numFields; i++ {
		field := objType.Field(i)
		columnMapper, err := NewColumnMapper(objType, i, field, params)
		if err != nil {
			return nil, nil, err
		}
		if columnMapper == nil {
			continue
		}

		fieldsMapper[i] = columnMapper
		if err = pTree.Put(i, columnMapper.titlePath); err != nil {
			return nil, nil, err
		}
	}

	// 构建 fieldIndex -> []columnIndex 的映射
	// 1. 先从PathTree获取 columnIndex -> fieldIndex
	columnToField, err := titlePathMatcher(pTree, params)
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
		// 当一个类型属性映射多列数据时，对象属性类型必须是slice。
		// 而当类型属性只映射一列数据时，对象属性类型可以是slice，也可以非slice。当是slice时，传入setter的参数是长度为1的slice值。
		if len(columnIndexes) > 1 {
			if !columnMapper.setter.mappingType.IsSlice() {
				return nil, nil, fmt.Errorf("eorm: a slice mapping type is needed for multi-columns at field index %d", fieldIndex)
			}
		}

		sort.Ints(columnIndexes)
	}
	// 4. 检查constraint required是否满足
	for fieldIndex, columnMapper := range fieldsMapper {
		if columnMapper.constraint.NeedMapper() {
			columnIndexes := fieldToColumns[fieldIndex]
			if len(columnIndexes) == 0 {
				return nil, nil, fmt.Errorf("%w for %q field at index %d",
					ErrRequiredColumnNotFound, columnMapper.constraint.String(), fieldIndex)
			}
		}
	}
	mp := &RowMapper[T]{
		typ:     objType,
		params:  params,
		fields:  fieldsMapper,
		columns: fieldToColumns,
	}
	// 5. 检查 match level
	switch params.RequiredMatchLevel.Formalize() {
	case MatchLevelPerfect:
		if !mp.IsPerfectMatch() {
			return nil, nil, ErrInsufficientMatchLevel
		}
	case MatchLevelMatched:
		if !mp.IsMatched() {
			return nil, nil, ErrInsufficientMatchLevel
		}
	default:
		// ok
	}

	return mp, pTree, nil
}

func NewRowMapper[T any](objType reflect.Type, sheet Sheet, params *Params) (*RowMapper[T], *PathTree[int], error) {
	return newRowMapper[T](
		objType,
		func(tree *PathTree[int], params *Params) (columnToField map[int]int, err error) {
			return MatchTitlePath(tree, sheet, params)
		},
		params,
	)
}

func NewRowMapperByStream[T any](objType reflect.Type, stream StreamSheet, params *Params) (*RowMapper[T], *PathTree[int], error) {
	return newRowMapper[T](
		objType,
		func(tree *PathTree[int], params *Params) (columnToField map[int]int, err error) {
			return MatchTitlePathByStream(tree, stream, params)
		},
		params,
	)
}

// IsPerfectMatch 对象每一个属性都找到了对应列
func (m *RowMapper[T]) IsPerfectMatch() bool {
	return len(m.fields) > 0 && len(m.fields) == len(m.columns)
}

// IsMatched 对象中至少有一个属性找到了对应列
func (m *RowMapper[T]) IsMatched() bool { return len(m.columns) > 0 }

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
		if err := columnMapper.CellToField(val, row, columnIndexes, m.params); err != nil {
			return nil, err
		}
	}
	return val.Interface().(*T), nil
}

// fieldValueSlice 将val转为string/int64/float64/bool这4种类型之一的slice
func fieldValueSlice[T string | int64 | float64 | bool](val reflect.Value, valueOf func(elemVal reflect.Value) T) []T {
	length := val.Len()
	values := make([]T, length)
	for i := 0; i < length; i++ {
		values[i] = valueOf(val.Index(i))
	}
	return values
}

func writeFieldToCells[T string | int64 | float64 | bool](fieldValues []T, columnIndexes []int, w SheetWriter, rowIndex int) error {
	var err error
	for i := 0; i < len(fieldValues) && i < len(columnIndexes); i++ {
		err = w.Write(rowIndex, columnIndexes[i], fieldValues[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *RowMapper[T]) Write(w SheetWriter, rowIndex int, obj *T) error {
	if obj == nil {
		return nil
	}
	if rowIndex < 0 {
		return fmt.Errorf("eorm: invalid row index: %d", rowIndex)
	}
	objVal := reflect.ValueOf(obj)
	for fieldIndex, columnIndexes := range m.columns {
		mapper := m.fields[fieldIndex]
		if mapper == nil {
			return fmt.Errorf("eorm: no column mapper found for field index %d", fieldIndex)
		}
		fieldVal, err := mapper.fieldValue(objVal)
		if err != nil {
			return err
		}
		if len(columnIndexes) == 0 {
			continue
		}
		switch mapper.getter.mappingType {
		case MTStringSlice:
			ss := fieldValueSlice[string](fieldVal, func(elemVal reflect.Value) string { return elemVal.String() })
			err = writeFieldToCells(ss, columnIndexes, w, rowIndex)
		case MTInt64Slice:
			is := fieldValueSlice[int64](fieldVal, func(elemVal reflect.Value) int64 { return elemVal.Int() })
			err = writeFieldToCells(is, columnIndexes, w, rowIndex)
		case MTFloat64Slice:
			fs := fieldValueSlice[float64](fieldVal, func(elemVal reflect.Value) float64 { return elemVal.Float() })
			err = writeFieldToCells(fs, columnIndexes, w, rowIndex)
		case MTBoolSlice:
			bs := fieldValueSlice[bool](fieldVal, func(elemVal reflect.Value) bool { return elemVal.Bool() })
			err = writeFieldToCells(bs, columnIndexes, w, rowIndex)
		case MTString:
			err = w.Write(rowIndex, columnIndexes[0], fieldVal.String())
		case MTInt64:
			err = w.Write(rowIndex, columnIndexes[0], fieldVal.Int())
		case MTFloat64:
			err = w.Write(rowIndex, columnIndexes[0], fieldVal.Float())
		case MTBool:
			err = w.Write(rowIndex, columnIndexes[0], fieldVal.Bool())
		default:
			return fmt.Errorf("eorm: unknown field mapping type at index %d: %s", fieldIndex, mapper.getter.mappingType)
		}
	}
	return nil
}
