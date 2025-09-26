package eorm

import (
	"fmt"
	"io"
	"iter"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

type (
	xlsxWorkbook struct {
		names []string
		f     *excelize.File
	}

	xlsxSheet struct {
		name    string
		allRows [][]string
	}

	xlsxRowIterator struct {
		rows *excelize.Rows
	}

	xlsxRow []string
)

func NewXlsxWorkbook(filePath string) (Workbook, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("excel/xlsx: %w", err)
	}
	names := f.GetSheetList()
	return &xlsxWorkbook{names: names, f: f}, nil
}

func NewXlsxWorkbookByReadSeeker(reader io.ReadSeeker) (Workbook, error) {
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("excel/xlsx: %w", err)
	}
	names := f.GetSheetList()
	return &xlsxWorkbook{names: names, f: f}, nil
}

func (x *xlsxWorkbook) SheetCount() int {
	return len(x.names)
}

func (x *xlsxWorkbook) GetSheet(index int) (Sheet, error) {
	if index < 0 || index >= len(x.names) {
		return nil, ErrOutOfRange
	}
	return x.GetSheetByName(x.names[index])
}

func (x *xlsxWorkbook) GetSheetByName(name string) (Sheet, error) {
	rows, err := x.f.GetRows(name)
	if err != nil {
		return nil, fmt.Errorf("excel/xlsx: %w", err)
	}
	return &xlsxSheet{name: name, allRows: rows}, nil
}

func (x *xlsxWorkbook) IterateSheet(index int) (RowIterator, error) {
	if index < 0 || index >= len(x.names) {
		return nil, ErrOutOfRange
	}
	rows, err := x.f.Rows(x.names[index])
	if err != nil {
		return nil, fmt.Errorf("excel/xlsx: %w", err)
	}
	return &xlsxRowIterator{rows: rows}, nil
}

func (x *xlsxWorkbook) Close() error {
	err := x.f.Close()
	if err != nil {
		return fmt.Errorf("excel/xlsx: %w", err)
	}
	return nil
}

func (x xlsxSheet) GetName() string {
	return x.name
}

func (x xlsxSheet) RowCount() int {
	return len(x.allRows)
}

func (x xlsxSheet) GetRow(index int) (Row, error) {
	if index < 0 || index >= len(x.allRows) {
		return nil, ErrOutOfRange
	}
	return xlsxRow(x.allRows[index]), nil
}

func (x xlsxRow) ColumnCount() int {
	return len(x)
}

func (x xlsxRow) GetColumn(index int) (string, error) {
	if index < 0 || index >= len(x) {
		return "", ErrOutOfRange
	}
	return x[index], nil
}

func (x xlsxRow) GetInt64Column(index int) (int64, error) {
	v, err := x.GetColumn(index)
	if err != nil {
		return 0, err
	}
	if v == "" {
		return 0, ErrEmptyCell
	}

	i, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("excel/xlsx: string to int64 %w: %w", ErrParseError, err)
	}
	return i, nil
}

func (x xlsxRow) GetFloat64Column(index int) (float64, error) {
	v, err := x.GetColumn(index)
	if err != nil {
		return 0, err
	}
	if v == "" {
		return 0, ErrEmptyCell
	}

	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return 0, fmt.Errorf("excel/xlsx: string to float64 %w: %w", ErrParseError, err)
	}
	return f, nil
}

func (x xlsxRow) GetBoolColumn(index int) (bool, error) {
	v, err := x.GetColumn(index)
	if err != nil {
		return false, err
	}
	if v == "" {
		return false, ErrEmptyCell
	}

	v = strings.ToUpper(v)
	switch v {
	case "TRUE":
		return true, nil
	case "FALSE":
		return false, nil
	default:
		return false, fmt.Errorf("excel/xlsx: string to bool %w: unknown value: %s", ErrParseError, v)
	}
}

func (x xlsxRow) AllColumns() iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		for i, s := range x {
			if !yield(i, s) {
				return
			}
		}
	}
}

func (x xlsxRowIterator) Next() bool {
	return x.rows.Next()
}

func (x xlsxRowIterator) Current() (Row, error) {
	row, err := x.rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("excel/xlsx: %w", err)
	}
	return xlsxRow(row), nil
}

func (x xlsxRowIterator) Close() error {
	return x.rows.Close()
}
