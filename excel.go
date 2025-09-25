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
	Row interface {
		ColumnCount() int
		GetColumn(index int) (string, error)
		GetInt64Column(index int) (int64, error)
		GetFloat64Column(index int) (float64, error)
		GetBoolColumn(index int) (bool, error)
		AllColumns() iter.Seq2[int, string]
	}

	Sheet interface {
		GetName() string
		RowCount() int
		GetRow(index int) (Row, error)
	}

	Workbook interface {
		SheetCount() int
		GetSheet(index int) (Sheet, error)
		IterateSheet(index int) (RowIterator, error)
		Close() error
	}

	RowIterator interface {
		Next() bool
		Current() (Row, error)
		Close() error
	}

	RowReader struct {
		Row
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
