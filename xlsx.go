package eorm

import (
	"fmt"
	"io"
	"iter"
	"strconv"
	"strings"

	"github.com/stephenfire/go-tools"
	"github.com/xuri/excelize/v2"
)

type (
	xlsxWorkbook struct {
		names   []string
		nameSet tools.KSet[string] // valid sheet names
		f       *excelize.File
	}

	xlsxSheet struct {
		name    string
		allRows [][]string
	}

	xlsxRowIterator struct {
		rows   *excelize.Rows
		rowNum int
	}

	xlsxRow []string

	xlsxWriter struct {
		f         *excelize.File
		sheetName string
	}
)

func NewXlsxWorkbookByFile(f *excelize.File) (Workbook, error) {
	names := f.GetSheetList()
	nameSet := tools.NewKSet[string](names...)
	return &xlsxWorkbook{names: names, nameSet: nameSet, f: f}, nil
}

func NewXlsxWorkbook(filePath string) (Workbook, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("excel/xlsx: %w", err)
	}
	return NewXlsxWorkbookByFile(f)
}

func NewXlsxWorkbookByReadSeeker(reader io.ReadSeeker) (Workbook, error) {
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("excel/xlsx: %w", err)
	}
	return NewXlsxWorkbookByFile(f)
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
	if !x.nameSet.IsExist(name) {
		return nil, ErrNotFound
	}
	rows, err := x.f.GetRows(name)
	if err != nil {
		return nil, fmt.Errorf("excel/xlsx: %w", err)
	}
	return &xlsxSheet{name: name, allRows: rows}, nil
}

func (x *xlsxWorkbook) GetStreamSheet(index int) (StreamSheet, error) {
	if index < 0 || index >= len(x.names) {
		return nil, ErrOutOfRange
	}
	return x.GetStreamSheetByName(x.names[index])
}

func (x *xlsxWorkbook) GetSheetWriter(index int) (SheetWriter, error) {
	if index < 0 || index >= len(x.names) {
		return nil, ErrOutOfRange
	}
	return &xlsxWriter{f: x.f, sheetName: x.names[index]}, nil
}

func (x *xlsxWorkbook) GetSheetWriterByName(name string) (SheetWriter, error) {
	if !x.nameSet.IsExist(name) {
		return nil, ErrNotFound
	}
	return &xlsxWriter{f: x.f, sheetName: name}, nil
}

func (x *xlsxWorkbook) GetStreamSheetByName(name string) (StreamSheet, error) {
	if !x.nameSet.IsExist(name) {
		return nil, ErrNotFound
	}
	return newXlsxRowIterator(x.f, name)
}

func (x *xlsxWorkbook) Save() error {
	if err := x.f.Save(); err != nil {
		return fmt.Errorf("excel/xlsx: %w", err)
	}
	return nil
}

func (x *xlsxWorkbook) WriteTo(w io.Writer) (int64, error) {
	n, err := x.f.WriteTo(w)
	if err != nil {
		return n, fmt.Errorf("excel/xlsx: %w", err)
	}
	return n, nil
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

func newXlsxRowIterator(f *excelize.File, sheetName string) (StreamSheet, error) {
	rows, err := f.Rows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("excel/xlsx: %w", err)
	}
	return &xlsxRowIterator{rows: rows, rowNum: -1}, nil
}

func (x *xlsxRowIterator) Next() bool {
	if x.rows.Next() {
		x.rowNum++
		return true
	}
	return false
}

func (x *xlsxRowIterator) Current() (Row, error) {
	row, err := x.rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("excel/xlsx: %w", err)
	}
	return xlsxRow(row), nil
}

func (x *xlsxRowIterator) CurrentRowNumber() int {
	return x.rowNum
}

func (x *xlsxRowIterator) Skip(rowCount int) error {
	for i := 0; i < rowCount; i++ {
		if x.Next() {
			continue
		}
		return ErrOutOfRange
	}
	return nil
}

func (x *xlsxRowIterator) Close() error {
	return x.rows.Close()
}

func (x *xlsxWriter) Write(row, column int, value any) error {
	col, err := excelize.ColumnNumberToName(column + 1)
	if err != nil {
		return fmt.Errorf("excel/xlsx: %w", err)
	}
	cell := fmt.Sprintf("%s%d", col, row+1)
	return x.f.SetCellValue(x.sheetName, cell, value)
}

func (x *xlsxWriter) SaveTo(w io.Writer) error {
	_, err := x.f.WriteTo(w)
	if err != nil {
		return fmt.Errorf("excel/xlsx: %w", err)
	}
	return nil
}

func (x *xlsxWriter) Close() error {
	if err := x.f.Close(); err != nil {
		return fmt.Errorf("excel/xlsx: %w", err)
	}
	return nil
}

func (x *xlsxWriter) Stream() (StreamSheet, error) {
	return newXlsxRowIterator(x.f, x.sheetName)
}
