package eorm

import (
	"errors"
	"fmt"
	"io"
	"iter"
	"path/filepath"
	"strings"
)

type (
	// Row excel表格中一行数据的抽象类型接口
	Row interface {
		ColumnCount() int
		GetColumn(index int) (string, error)
		GetInt64Column(index int) (int64, error)
		GetFloat64Column(index int) (float64, error)
		GetBoolColumn(index int) (bool, error)
		AllColumns() iter.Seq2[int, string]
	}

	// Sheet excel中一个表格的抽象类型接口，可随机读取表格中的行内容
	Sheet interface {
		GetName() string
		RowCount() int
		GetRow(index int) (Row, error)
	}

	// Workbook excel文件的抽象接口类型
	Workbook interface {
		SheetCount() int
		GetSheet(index int) (Sheet, error)
		GetSheetByName(name string) (Sheet, error)
		GetStreamSheet(index int) (StreamSheet, error)
		GetStreamSheetByName(name string) (StreamSheet, error)
		GetSheetWriter(index int) (SheetWriter, error)
		GetSheetWriterByName(name string) (SheetWriter, error)
		Close() error
	}

	// StreamSheet 通过stream模式读取row data的Sheet，在使用完打开的 StreamSheet 之前不应关闭 Workbook。只能按顺序遍历。
	StreamSheet interface {
		Next() bool
		Current() (Row, error)
		CurrentRowNumber() int
		Skip(rowCount int) error
		Close() error
	}

	// SheetWriter used to write ORM object to excel sheet
	SheetWriter interface {
		// Write write value to the cell(row, column), where both row and column start with 0.
		// The valid value types are int64, float64, string, and bool.
		Write(row, column int, value any) error
		// Stream get StreamSheet of current sheet
		Stream() (StreamSheet, error)
	}
)

var (
	ErrNotFound         = errors.New("excel: not found")
	ErrOutOfRange       = errors.New("excel: out of range")
	ErrNil              = errors.New("excel: nil")
	ErrEmptyCell        = errors.New("excel: empty cell")
	ErrInvalidValueType = errors.New("excel: invalid value type")
	// ErrInvalidCellValue #NULL!, #DIV/0!, #VALUE!, #REF!, #NAME?, #NUM!!, #N/A
	ErrInvalidCellValue    = errors.New("excel: invalid cell value")
	ErrExcelNotInitialized = errors.New("excel: not initialized")
	ErrEof                 = errors.New("excel: eof")
	ErrParseError          = errors.New("cell value parse error")
)

func NewWorkbook(filePath string) (Workbook, error) {
	// 根据文件扩展名选择合适的Workbook实现
	var wb Workbook
	var err error
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".xlsx":
		wb, err = NewXlsxWorkbook(filePath)
	case ".xls":
		wb, err = NewXlsWorkbook(filePath)
	default:
		return nil, fmt.Errorf("eorm: unsupported file format: %s", ext)
	}
	if err != nil {
		return nil, fmt.Errorf("eorm: failed to open workbook: %w", err)
	}
	return wb, nil
}

func NewWorkbookByReadSeeker(filename string, reader io.ReadSeeker) (Workbook, error) {
	var wb Workbook
	var err error
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".xlsx":
		wb, err = NewXlsxWorkbookByReadSeeker(reader)
	case ".xls":
		wb, err = NewXlsWorkbookByReadSeeker(reader)
	default:
		return nil, fmt.Errorf("eorm: unsupported file format: %s", ext)
	}
	if err != nil {
		return nil, fmt.Errorf("eorm: failed to open workbook: %w", err)
	}
	return wb, nil
}
