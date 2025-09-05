package eorm

import (
	"fmt"
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
		allRows [][]string
	}

	xlsxRow []string
)

func NewXlsxWorkbook(filePath string) (Workbook, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("eorm/excelize: %w", err)
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
	rows, err := x.f.GetRows(x.names[index])
	if err != nil {
		return nil, fmt.Errorf("eorm/excelize: %w", err)
	}
	return &xlsxSheet{allRows: rows}, nil
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

	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("eorm: parse xlsx string to int64 failed: %w", err)
	}
	return i, nil
}

func (x xlsxRow) GetFloat64Column(index int) (float64, error) {
	v, err := x.GetColumn(index)
	if err != nil {
		return 0, err
	}

	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("eorm: parse xlsx string to float64 failed: %w", err)
	}
	return f, nil
}

func (x xlsxRow) GetBoolColumn(index int) (bool, error) {
	v, err := x.GetColumn(index)
	if err != nil {
		return false, err
	}

	v = strings.ToUpper(v)
	switch v {
	case "TRUE":
		return true, nil
	case "FALSE":
		return false, nil
	default:
		return false, fmt.Errorf("eorm: parse xlsx string to bool failed: unknown value: %s", v)
	}
}
