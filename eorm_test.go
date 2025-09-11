package eorm

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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

func TestNewEORM(t *testing.T) {
	objType := reflect.TypeOf(TestUser{})

	// 测试创建EORM实例
	eorm, err := NewEORM[TestUser](filepath.Join("testdata", "title.xlsx"), objType)
	if err != nil {
		t.Fatalf("NewEORM failed: %v", err)
	}
	defer func() {
		_ = eorm.Close()
	}()

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

func TestAnalyzeObjectType(t *testing.T) {
	objType := reflect.TypeOf(TestUser{})

	rowMapper, err := analyzeObjectType(objType)
	if err != nil {
		t.Fatalf("analyzeObjectType failed: %v", err)
	}

	// 验证字段数量
	expectedFieldCount := 5 // TestUser有5个带eorm标签的字段
	if len(rowMapper.fields) != expectedFieldCount {
		t.Errorf("Expected %d fields, got %d", expectedFieldCount, len(rowMapper.fields))
	}

	// 验证每个字段的ColumnMapper
	for fieldIndex, columnMapper := range rowMapper.fields {
		if columnMapper == nil {
			t.Errorf("ColumnMapper for field index %d is nil", fieldIndex)
			continue
		}

		field := objType.Field(fieldIndex)

		// 验证字段名匹配
		if columnMapper.fieldName != field.Name {
			t.Errorf("Field name mismatch: expected %s, got %s", field.Name, columnMapper.fieldName)
		}

		// 验证字段索引匹配
		if columnMapper.fieldIndex != fieldIndex {
			t.Errorf("Field index mismatch: expected %d, got %d", fieldIndex, columnMapper.fieldIndex)
		}

		// 验证titlePath解析正确
		eormTag := field.Tag.Get("eorm")
		expectedTitlePath, err := TitlePath(nil).Decode(eormTag)
		if err != nil {
			t.Errorf("Failed to decode expected title path: %v", err)
			continue
		}

		if len(columnMapper.titlePath) != len(expectedTitlePath) {
			t.Errorf("TitlePath length mismatch for field %s: expected %d, got %d",
				field.Name, len(expectedTitlePath), len(columnMapper.titlePath))
		}

		for i, part := range expectedTitlePath {
			if columnMapper.titlePath[i] != part {
				t.Errorf("TitlePath part mismatch for field %s at index %d: expected %s, got %s",
					field.Name, i, part, columnMapper.titlePath[i])
			}
		}
	}
}

func TestFindSetterMethod(t *testing.T) {
	objType := reflect.TypeOf(TestUser{})

	// 测试Email字段的setter方法
	method, isSlice, found := findSetterMethod(objType, "Email")
	if !found {
		t.Fatal("Should find setter method for Email field")
	}
	if method.Name != "SetEmail" {
		t.Fatalf("Expected method name SetEmail, got %s", method.Name)
	}
	if isSlice {
		t.Fatal("Should not be a slice setter method for Email field")
	}

	// 测试Name字段的数组setter方法
	method, isSlice, found = findSetterMethod(objType, "Names")
	if !found {
		t.Error("Should find array setter method for Names field")
	}
	if method.Name != "SetNames" {
		t.Errorf("Expected method name SetNames, got %s", method.Name)
	}
	if !isSlice {
		t.Fatal("Should be a slice setter method for Names field")
	}

	// 测试不存在的字段
	method, isSlice, found = findSetterMethod(objType, "NonExistent")
	if found {
		t.Error("Should not find setter method for non-existent field")
	}

	// 测试没有setter方法的字段
	method, isSlice, found = findSetterMethod(objType, "Age")
	if found {
		t.Error("Should not find setter method for Age field")
	}
}

func TestBuildColumnTree(t *testing.T) {
	objType := reflect.TypeOf(TestUser{})
	rowMapper, err := analyzeObjectType(objType)
	if err != nil {
		t.Fatalf("analyzeObjectType failed: %v", err)
	}

	tree, err := buildColumnTree(rowMapper)
	if err != nil {
		t.Fatalf("buildColumnTree failed: %v", err)
	}

	// 验证树深度
	depth, err := tree.Check()
	if err != nil {
		t.Fatalf("Tree check failed: %v", err)
	}

	// TestUser的title path深度应该都是3层（因为有//表示3层）
	expectedDepth := 3
	if depth != expectedDepth {
		t.Errorf("Expected tree depth %d, got %d", expectedDepth, depth)
	}
}

func TestNewEORM_UnsupportedFileFormat(t *testing.T) {
	objType := reflect.TypeOf(TestUser{})

	// 测试不支持的文件格式
	_, err := NewEORM[TestUser]("test.txt", objType)
	if err == nil {
		t.Error("Should return error for unsupported file format")
	}

	expectedError := "unsupported file format"
	if err != nil && !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Error should contain '%s', got: %v", expectedError, err)
	}
}

func TestNewEORM_NonStructType(t *testing.T) {
	// 测试非结构体类型
	_, err := NewEORM[string]("test.xlsx", reflect.TypeOf("string"))
	if err == nil {
		t.Error("Should return error for non-struct type")
	}

	expectedError := "objType must be a struct"
	if err != nil && !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Error should contain '%s', got: %v", expectedError, err)
	}
}
