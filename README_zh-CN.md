# go-eorm: Go语言的Excel对象关系映射库

[English Documentation](README.md)

## 概述

go-eorm 是一个强大的 Go 语言 Excel 对象关系映射库，提供 Excel 文件和 Go 结构体之间的无缝映射。它支持 `.xls` 和 `.xlsx` 两种文件格式，并为复杂的 Excel 数据结构提供灵活的配置选项。

## 特性

- **双格式支持**: 使用不同的底层库读取 `.xls` 和 `.xlsx` 文件
- **分层标题映射**: 支持使用路径式语法处理多级 Excel 表头
- **灵活类型映射**: 内置支持 `string`、`int64`、`float64` 和 `bool` 类型
- **自定义设置器**: 为复杂类型定义自定义解析逻辑
- **数组映射**: 处理具有相同标题路径的多个列
- **约束验证**: 支持 `required` 和 `not_null` 约束
- **合并单元格处理**: 智能处理表头中的合并单元格
- **字符转义**: 自动转义标题路径中的特殊字符

## 安装

```bash
go get github.com/stephenfire/go-eorm
```

## 快速开始

### 基础用法

```go
package main

import (
    "reflect"
    "github.com/stephenfire/go-eorm"
)

// 使用 eorm 标签定义结构体
type User struct {
    ID     int64  `eorm:"序号//"`
    Name   string `eorm:"名称//"`
    Email  string `eorm:"第一级/邮箱/地址"`
    Age    int64  `eorm:"第一级/年龄/数值"`
    Active bool   `eorm:"状态//"`
}

func main() {
    // 打开 Excel 文件
    wb, err := eorm.NewWorkbook("users.xlsx")
    if err != nil {
        panic(err)
    }
    defer wb.Close()

    // 获取第一个工作表
    sheet, err := wb.GetSheet(0)
    if err != nil {
        panic(err)
    }

    // 创建 EORM 实例
    eorm, err := eorm.NewEORM[User](sheet, reflect.TypeOf(User{}))
    if err != nil {
        panic(err)
    }

    // 遍历行数据
    for eorm.Next() {
        user, err := eorm.Current()
        if err != nil {
            panic(err)
        }
        fmt.Printf("用户: %+v\n", user)
    }
}
```

### 使用自定义设置器的高级示例

```go
type AdvancedUser struct {
    ID     int64    `eorm:"序号//"`
    Name   string   `eorm:"名称//"`
    Emails []string `eorm:"第一级/邮箱/地址"`
    Tags   []string `eorm:"标签//"`
}

// 邮箱的自定义设置器
func (u *AdvancedUser) SetEmails(emails []string) {
    u.Emails = make([]string, len(emails))
    for i, email := range emails {
        u.Emails[i] = "processed_" + email
    }
}

// 标签的自定义设置器
func (u *AdvancedUser) SetTags(tags []string) {
    u.Tags = tags
}
```

## 标题路径系统

### 基础语法

标题路径使用 `/` 作为分隔符来表示分层的 Excel 表头：

```go
type Example struct {
    Simple   string `eorm:"简单标题"`           // 单级表头
    Nested   string `eorm:"第一级/第二级"`      // 两级表头  
    Deep     string `eorm:"第一级/第二级/第三级"` // 三级表头
}
```

### 特殊字符和转义

标题路径中的特殊字符必须使用 `%HH` 格式进行转义：

| 字符 | 转义序列 |
|------|-------|
| `%`       | `%25` |
| `'`       | `%27` |
| `,`       | `%2C` |
| `"`       | `%22` |
| `/`       | `%2F` |
| `\`       | `%5C` |
| `\n`      | `%0A` |
| `\r`      | `%0D` |
| `\t`      | `%09` |
| `` ` ``   | `%60` |
| Space     | `%20` |

示例：
```go
type SpecialChars struct {
    Slash   string `eorm:"第一级/斜杠%2F测试"`
    Backslash string `eorm:"第一级/反斜杠%5C测试"`
    Backtick string `eorm:"第一级/反引号%60测试"`
    Quote   string `eorm:"第一级/双引号%22测试"`
    Space   string `eorm:"第一级/空%20格"`
}
```

### 通配第一行

使用空的第一个标题来跳过第一行表头：

```go
type WildcardExample struct {
    Field string `eorm:"/第二级标题"` // 跳过第一行，匹配第二行
}
```

## 约束

### Required 约束

确保列必须存在于 Excel 文件中：

```go
type RequiredExample struct {
    ID   int64  `eorm:"ID,required"`     // 列必须存在
    Name string `eorm:"名称,required"`    // 列必须存在
}
```

### Not Null 约束

确保列存在且包含非空值：

```go
type NotNullExample struct {
    ID   int64  `eorm:"ID,not_null"`     // 列必须存在且有值
    Name string `eorm:"名称,not_null"`    // 列必须存在且有值
}
```

## 配置选项

### 可用选项

```go
// 使用选项创建 EORM
eorm, err := eorm.NewEORM[User](sheet, reflect.TypeOf(User{}),
    eorm.WithTrimSpace(),              // 去除字符串两端的空白字符
    eorm.WithIgnoreOutOfRange(),       // 忽略越界错误
    eorm.WithIgnoreParseError(),       // 忽略解析错误
    eorm.WithIgnoreReadRowError(),     // 忽略行读取错误
    eorm.WithFirstRowWildcard(),       // 为第一行生成通配符
    eorm.WithGenLastLayerNoMerged(),   // 生成未合并的最后一层
    eorm.WithTitleStartRow(2),         // 从第2行开始读取标题
    eorm.WithMatchLevel(eorm.Strict),  // 设置匹配级别
)
```

### 匹配级别

- `eorm.MatchLevelNone`: 标准匹配（默认）
- `eorm.MatchLevelMatched`: 至少部分匹配
- `eorm.MatchLevelPerfect`: 要求精确匹配

## 错误处理

### 常见错误

```go
eorm, err := eorm.NewEORM[User](sheet, reflect.TypeOf(User{}))
if err != nil {
    // 处理初始化错误
}

for eorm.Next() {
    user, err := eorm.Current()
    if err != nil {
        // 处理行处理错误
        if errors.Is(err, eorm.ErrRequiredColumnNotFound) {
            // 处理缺失的必需列
        }
        if errors.Is(err, eorm.ErrInvalidState) {
            // 处理无效状态
        }
    }
}

if eorm.LastError() != nil {
    // 处理最后一个错误
}
```

## 文件格式支持

### XLS 文件
- 使用 `github.com/shakinm/xlsReader` 库
- 支持较旧的 Excel 格式 (.xls)

### XLSX 文件  
- 使用 `github.com/xuri/excelize/v2` 库
- 支持现代 Excel 格式 (.xlsx)

## 性能考虑

- 库使用反射进行映射，但会缓存映射以提高性能
- 对于大文件，考虑分批处理行数据
- 使用适当的匹配级别来平衡性能和准确性

## 测试

包包含全面的测试。运行测试：

```bash
go test ./...
```

## 许可证

版权所有 2025 stephen.fire@gmail.com

## 贡献

欢迎贡献！请随时提交拉取请求或为错误和功能请求打开问题。
