package eorm

import (
	"fmt"
	"reflect"
	"testing"
)

func TestNameEncode(t *testing.T) {
	tests := []struct {
		in, out string
	}{
		{in: "反引号`测试", out: "反引号%60测试"},
		{in: "空 格", out: "空%20格"},
		{in: "斜杠/", out: "斜杠%2F"},
		{in: "双引号\"测试", out: "双引号%22测试"},
		{in: "反斜杠\\", out: "反斜杠%5C"},
		{in: "序号", out: "序号"},
	}

	for _, test := range tests {
		got := TitleEscape(test.in)
		if got != test.out {
			t.Fatalf("TitleEscape(%q) = %q, want %q", test.in, got, test.out)
		}
		un, err := TitleUnescape(got)
		if err != nil {
			t.Fatalf("TitleUnescape(%q): %v", got, err)
		}
		if un != test.in {
			t.Fatalf("TitleUnescape(%q) = %q, want %q", got, un, test.in)
		}
		fmt.Printf("[%s] => [%s] <= [%s]\n", test.in, got, un)
	}
	t.Log("TestNameEncode OK")
}

// User 定义一个示例结构体，包含各种标签
type User struct {
	ID     int    `json:"id" validate:"required"`
	Name   string `json:"name" validate:"required,min=2,max=50"`
	Email  string `json:"email" validate:"required,email"`
	Age    int    `json:"age,omitempty" validate:"gte=0,lte=150"`
	Active bool   `json:"active"`
}

// GetStructTags 获取结构体字段的所有标签信息
// 参数:
//
//	s: 任意结构体变量
//
// 返回:
//
//	一个 map，键是字段名，值是另一个 map，包含标签的键值对。
//	例如: map["ID"] = map["json"]="id", "validate"="required"
func GetStructTags(s interface{}) (map[string]map[string]string, error) {
	// 获取 s 的反射值
	v := reflect.ValueOf(s)

	// 如果 s 是指针，则获取其指向的元素
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// 检查 v 是否为结构体
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("输入的参数不是一个结构体或结构体指针")
	}

	// 获取 s 的反射类型
	t := v.Type()
	tagsMap := make(map[string]map[string]string)

	// 遍历结构体的所有字段
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tags := make(map[string]string)

		// 遍历字段的所有标签
		for _, tag := range []string{"json", "validate"} { // 你可以根据需要添加更多标签
			if val, ok := field.Tag.Lookup(tag); ok {
				tags[tag] = val
			}
		}

		// 如果标签不为空，则添加到结果中
		if len(tags) > 0 {
			tagsMap[field.Name] = tags
		}
	}

	return tagsMap, nil
}

func TestTags(t *testing.T) {
	user := User{
		ID:    1,
		Name:  "张三",
		Email: "zhangsan@example.com",
		Age:   30,
	}

	// 获取 User 结构体的标签信息
	tags, err := GetStructTags(user)
	if err != nil {
		t.Fatal(err)
	}

	// 打印结果
	t.Log("User 结构体的标签信息:")
	for fieldName, tagMap := range tags {
		t.Logf("  字段名: %s\n", fieldName)
		for tagName, tagValue := range tagMap {
			t.Logf("    - 标签: %s, 值: %s\n", tagName, tagValue)
		}
	}
}
