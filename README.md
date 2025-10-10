# go-eorm: Excel Object Relational Mapping for Go

[中文文档](README_zh-CN.md)

## Overview

go-eorm is a powerful Excel Object Relational Mapping library for Go that provides seamless mapping between Excel files and Go structs. It supports both `.xls` and `.xlsx` file formats and offers flexible configuration options for complex Excel data structures.

## Features

- **Dual Format Support**: Read both `.xls` and `.xlsx` files using different underlying libraries
- **Hierarchical Title Mapping**: Support for multi-level Excel headers using path-like syntax
- **Flexible Type Mapping**: Built-in support for `string`, `int64`, `float64`, and `bool` types
- **Custom Setters**: Define custom parsing logic for complex types
- **Array Mapping**: Handle multiple columns with the same title path
- **Constraint Validation**: Support for `required` and `not_null` constraints
- **Merged Cell Handling**: Intelligent handling of merged cells in headers
- **Character Escaping**: Automatic escaping of special characters in title paths

## Installation

```bash
go get github.com/stephenfire/go-eorm
```

## Quick Start

### Basic Usage

```go
package main

import (
    "reflect"
    "github.com/stephenfire/go-eorm"
)

// Define your struct with eorm tags
type User struct {
    ID     int64  `eorm:"序号//"`
    Name   string `eorm:"名称//"`
    Email  string `eorm:"第一级/邮箱/地址"`
    Age    int64  `eorm:"第一级/年龄/数值"`
    Active bool   `eorm:"状态//"`
}

func main() {
    // Open Excel file
    wb, err := eorm.NewWorkbook("users.xlsx")
    if err != nil {
        panic(err)
    }
    defer wb.Close()

    // Get the first sheet
    sheet, err := wb.GetSheet(0)
    if err != nil {
        panic(err)
    }

    // Create EORM instance
    eorm, err := eorm.NewEORM[User](sheet, reflect.TypeOf(User{}))
    if err != nil {
        panic(err)
    }

    // Iterate through rows
    for eorm.Next() {
        user, err := eorm.Current()
        if err != nil {
            panic(err)
        }
        fmt.Printf("User: %+v\n", user)
    }
}
```

### Advanced Example with Custom Setters

```go
type AdvancedUser struct {
    ID     int64    `eorm:"序号//"`
    Name   string   `eorm:"名称//"`
    Emails []string `eorm:"第一级/邮箱/地址"`
    Tags   []string `eorm:"标签//"`
}

// Custom setter for emails
func (u *AdvancedUser) SetEmails(emails []string) {
    u.Emails = make([]string, len(emails))
    for i, email := range emails {
        u.Emails[i] = "processed_" + email
    }
}

// Custom setter for tags
func (u *AdvancedUser) SetTags(tags []string) {
    u.Tags = tags
}
```

## Title Path System

### Basic Syntax

Title paths use `/` as a delimiter to represent hierarchical Excel headers:

```go
type Example struct {
    Simple   string `eorm:"简单标题"`           // Single-level header
    Nested   string `eorm:"第一级/第二级"`      // Two-level header  
    Deep     string `eorm:"第一级/第二级/第三级"` // Three-level header
}
```

### Special Characters and Escaping

Special characters in title paths must be escaped using `%HH` format:

| Character | Escape Sequence |
|-----------|-----------------|
| `%`       | `%25`           |
| `'`       | `%27`           |
| `,`       | `%2C`           |
| `"`       | `%22`           |
| `/`       | `%2F`           |
| `\`       | `%5C`           |
| `\n`      | `%0A`           |
| `\r`      | `%0D`           |
| `\t`      | `%09`           |
| `` ` ``   | `%60`           |
| Space     | `%20`           |

Example:
```go
type SpecialChars struct {
    Slash   string `eorm:"第一级/斜杠%2F测试"`
    Backslash string `eorm:"第一级/反斜杠%5C测试"`
    Backtick string `eorm:"第一级/反引号%60测试"`
    Quote   string `eorm:"第一级/双引号%22测试"`
    Space   string `eorm:"第一级/空%20格"`
}
```

The `cmds/pathgener` tool can assist you in generating the necessary EORM tags for your Excel files.

### Wildcard First Row

Use an empty first title to skip the first header row:

```go
type WildcardExample struct {
    Field string `eorm:"/第二级标题"` // Skips first row, matches second row
}
```

## Constraints

### Required Constraint

Ensure that a column must exist in the Excel file:

```go
type RequiredExample struct {
    ID   int64  `eorm:"ID,required"`     // Column must exist
    Name string `eorm:"名称,required"`    // Column must exist
}
```

### Not Null Constraint

Ensure that a column exists and contains non-empty values:

```go
type NotNullExample struct {
    ID   int64  `eorm:"ID,not_null"`     // Column must exist and have values
    Name string `eorm:"名称,not_null"`    // Column must exist and have values
}
```

## Configuration Options

### Available Options

```go
// Create EORM with options
eorm, err := eorm.NewEORM[User](sheet, reflect.TypeOf(User{}),
    eorm.WithTrimSpace(),              // Trim whitespace from strings
    eorm.WithIgnoreOutOfRange(),       // Ignore out-of-range errors
    eorm.WithIgnoreParseError(),       // Ignore parsing errors
    eorm.WithIgnoreReadRowError(),     // Ignore row reading errors
    eorm.WithFirstRowWildcard(),       // Generate wildcard for first row
    eorm.WithGenLastLayerNoMerged(),   // Generate last layer without merged cells
    eorm.WithTitleStartRow(2),         // Start reading titles from row 2
    eorm.WithMatchLevel(eorm.MatchLevelPerfect),  // Set matching level
)
```

### Matching Levels

- `eorm.MatchLevelNone`: Standard matching (default)
- `eorm.MatchLevelMatched`: At least a partial match
- `eorm.MatchLevelPerfect`: Require exact matches

## Error Handling

### Common Errors

```go
eorm, err := eorm.NewEORM[User](sheet, reflect.TypeOf(User{}))
if err != nil {
    // Handle initialization errors
}

for eorm.Next() {
    user, err := eorm.Current()
    if err != nil {
        // Handle row processing errors
        if errors.Is(err, eorm.ErrRequiredColumnNotFound) {
            // Handle missing required columns
        }
        if errors.Is(err, eorm.ErrInvalidState) {
            // Handle invalid state
        }
    }
}

if eorm.LastError() != nil {
    // Handle last error
}
```

## File Format Support

### XLS Files
- Uses `github.com/shakinm/xlsReader` library
- Supports older Excel format (.xls)

### XLSX Files  
- Uses `github.com/xuri/excelize/v2` library
- Supports modern Excel format (.xlsx)

## Performance Considerations

- The library uses reflection for mapping but caches mappings for performance
- For large files, consider processing rows in batches
- Use appropriate matching levels to balance performance and accuracy

## Testing

The package includes comprehensive tests. Run tests with:

```bash
go test ./...
```

## License

Copyright 2025 stephen.fire@gmail.com

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues for bugs and feature requests.
