package eorm

import (
	"fmt"
	"math/big"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stephenfire/go-common/math"
	"github.com/stephenfire/go-tools"
)

// TestUser 测试用的结构体，包含各种eorm标签
type TestUser struct {
	ID     int    `eorm:"序号//"`
	Name   string `eorm:"名称//"`
	Email  string `eorm:"第一级/邮箱/地址"`
	Age    int    `eorm:"第一级/年龄/数值"`
	Active bool   `eorm:"状态//"`
}

// SetEmail 自定义setter方法
func (u *TestUser) SetEmail(email string) {
	u.Email = "custom_" + email
}

// SetNames 数组setter方法
func (u *TestUser) SetNames(names []string) {
	if len(names) > 0 {
		u.Name = names[0] + "_from_array"
	}
}

type Integer int64

type TitleObj1 struct {
	Id      int64     `eorm:"序号//"`
	Name    string    `eorm:"名称//"`
	Numbers []Integer `eorm:"第一级/第二级/第三级"`
	Bool    bool      `eorm:"第一级/反引号%60测试/空%20格"`
	Slash   *big.Int  `eorm:"第一级/双引号%22测试/反斜杠%5C"`
}

func (t *TitleObj1) SetSlash(in int64) {
	t.Slash = big.NewInt(in)
}

func (t *TitleObj1) Equals(o *TitleObj1) bool {
	if t == o {
		return true
	}
	if t == nil || o == nil {
		return false
	}
	return t.Id == o.Id && t.Name == o.Name &&
		tools.KS[Integer](t.Numbers).Equal(o.Numbers) &&
		t.Bool == o.Bool &&
		math.CompareBigInt(t.Slash, o.Slash) == 0
}

func (t *TitleObj1) String() string {
	if t == nil {
		return "<nil>"
	}
	return fmt.Sprintf("{id:%d name:%s numbers:%v bool:%t slash:%s}", t.Id, t.Name, t.Numbers, t.Bool, math.BigIntForPrint(t.Slash))
}

func TestTitle1(t *testing.T) {
	wb, err := NewWorkbook(filepath.Join("testdata", "title.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = wb.Close()
	}()
	if wb.SheetCount() == 0 {
		t.Fatal("eorm: sheet count is zero")
	}

	sheet, err := wb.GetSheet(0)
	if err != nil {
		t.Fatal(err)
	}
	// 测试创建EORM实例
	eorm, err := NewEORM[TitleObj1](sheet, reflect.TypeOf(TitleObj1{}))
	if err != nil {
		t.Fatalf("NewEORM failed: %v", err)
	}

	expectings := []*TitleObj1{
		&TitleObj1{Id: 10, Name: "name10", Numbers: []Integer{16, 17}, Bool: true, Slash: big.NewInt(14)},
		&TitleObj1{Id: 20, Name: "name20", Numbers: []Integer{26, 27}, Bool: false, Slash: big.NewInt(24)},
	}

	i := 0
	for eorm.Next() {
		rowObj, err := eorm.Current()
		if err != nil {
			t.Fatal(err)
		}
		if i >= len(expectings) {
			t.Fatalf("eorm: expected %d rows, got %d", len(expectings), i+1)
		}
		if !expectings[i].Equals(rowObj) {
			t.Fatalf("eorm: expected %+v, got %+v", expectings[i], rowObj)
		}
	}
}

func TestNewEORM(t *testing.T) {
	objType := reflect.TypeOf(TestUser{})

	wb, err := NewWorkbook(filepath.Join("testdata", "title.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = wb.Close()
	}()
	if wb.SheetCount() == 0 {
		t.Fatal("eorm: sheet count is zero")
	}

	sheet, err := wb.GetSheet(0)
	if err != nil {
		t.Fatal(err)
	}
	// 测试创建EORM实例
	eorm, err := NewEORM[TestUser](sheet, objType)
	if err != nil {
		t.Fatalf("NewEORM failed: %v", err)
	}

	// 验证EORM结构体字段
	if eorm.objType != objType {
		t.Errorf("Expected objType %v, got %v", objType, eorm.objType)
	}

	if eorm.rowMapper == nil {
		t.Error("Expected rowMapper to be initialized")
	}

	if eorm.columnTree == nil {
		t.Error("Expected columnTree to be initialized")
	}

	// 验证ColumnMapper是否正确创建
	expectedFields := 5 // TestUser有5个带eorm标签的字段
	if len(eorm.rowMapper.fields) != expectedFields {
		t.Errorf("Expected %d fields in rowMapper, got %d", expectedFields, len(eorm.rowMapper.fields))
	}

	// 验证特定字段的ColumnMapper
	for fieldIndex, columnMapper := range eorm.rowMapper.fields {
		if columnMapper == nil {
			t.Errorf("ColumnMapper for field index %d is nil", fieldIndex)
			continue
		}

		// 验证fieldIndex和fieldName匹配
		field := objType.Field(fieldIndex)
		if columnMapper.fieldName != field.Name {
			t.Errorf("Field name mismatch for index %d: expected %s, got %s",
				fieldIndex, field.Name, columnMapper.fieldName)
		}

		// 验证titlePath不为空
		if len(columnMapper.titlePath) == 0 {
			t.Errorf("TitlePath for field %s is empty", field.Name)
		}

		t.Logf("Field %d: Mapper: %s", fieldIndex, columnMapper)
	}

	// 验证setter方法检测
	emailFieldIndex := -1
	for i := 0; i < objType.NumField(); i++ {
		if objType.Field(i).Name == "Email" {
			emailFieldIndex = i
			break
		}
	}

	if emailFieldIndex != -1 {
		emailMapper := eorm.rowMapper.fields[emailFieldIndex]
		if emailMapper != nil && !emailMapper.HasSetter {
			t.Error("Email field should have setter method detected")
		}
	}
}
