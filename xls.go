package eorm

import (
	"fmt"
	"io"
	"iter"
	"strconv"
	"strings"

	"github.com/shakinm/xlsReader/xls"
	"github.com/shakinm/xlsReader/xls/record"
	"github.com/shakinm/xlsReader/xls/structure"
)

const (
	XlsCellBlank     XlsCellType = iota // empty cell
	XlsCellBoolOrErr                    // boolean value or error
	XlsCellFake                         // fake, not exist. like a placeholder
	XlsCellString                       // string value
	XlsCellFloat                        // float64
	XlsCellInt                          // int64
	XlsCellNil
	XlsCellUnknown
)

type (
	XlsCellType byte
	XlsCell     struct{}
)

func (x XlsCell) Type(data structure.CellData) XlsCellType {
	if data == nil {
		return XlsCellNil
	}
	switch data.(type) {
	case *record.Blank:
		return XlsCellBlank
	case *record.BoolErr:
		return XlsCellBoolOrErr
	case *record.FakeBlank:
		return XlsCellFake
	case *record.LabelBIFF8, *record.LabelBIFF5, *record.LabelSSt:
		return XlsCellString
	case *record.Number:
		return XlsCellFloat
	case *record.Rk:
		return XlsCellInt
	default:
		return XlsCellUnknown
	}
}

func (x XlsCell) IsBlank(data structure.CellData) bool  { return x.Type(data) == XlsCellBlank }
func (x XlsCell) IsBool(data structure.CellData) bool   { return x.Type(data) == XlsCellBoolOrErr }
func (x XlsCell) IsString(data structure.CellData) bool { return x.Type(data) == XlsCellString }
func (x XlsCell) IsFloat(data structure.CellData) bool  { return x.Type(data) == XlsCellFloat }
func (x XlsCell) IsInt(data structure.CellData) bool    { return x.Type(data) == XlsCellInt }

func (x XlsCell) ToString(data structure.CellData) (string, error) {
	if data == nil {
		return "", ErrNil
	}
	s := data.GetString()
	return s, nil
}

func (x XlsCell) ToFloat64(data structure.CellData) (float64, error) {
	switch x.Type(data) {
	case XlsCellBlank:
		return 0, ErrEmptyCell
	case XlsCellBoolOrErr:
		return 0, ErrInvalidValueType
	case XlsCellFake:
		return 0, ErrNotFound
	case XlsCellString, XlsCellFloat, XlsCellInt:
		s := strings.TrimSpace(data.GetString())
		if s == "" {
			return 0, nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("excel/xls: string to float64 %w: %w", ErrParseError, err)
		}
		return f, nil
	// case XlsCellFloat:
	// 	return data.GetFloat64(), nil
	// case XlsCellInt:
	// 	return float64(data.GetInt64()), nil
	case XlsCellNil:
		return 0, ErrNil
	default:
		return 0, fmt.Errorf("excel/xls: unknown cell type: %v", data.GetType())
	}
}

func (x XlsCell) ToInt64(data structure.CellData) (int64, error) {
	switch x.Type(data) {
	case XlsCellBlank:
		return 0, ErrEmptyCell
	case XlsCellBoolOrErr:
		return 0, ErrInvalidValueType
	case XlsCellFake:
		return 0, ErrNotFound
	case XlsCellString, XlsCellFloat, XlsCellInt:
		s := strings.TrimSpace(data.GetString())
		if s == "" {
			return 0, nil
		}
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("excel/xls: string to int64 %w: %w", ErrParseError, err)
		}
		return i, nil
	// case XlsCellFloat:
	// 	return 0, ErrInvalidValueType
	// case XlsCellInt:
	// 	return data.GetInt64(), nil
	case XlsCellNil:
		return 0, ErrNil
	default:
		return 0, fmt.Errorf("excel/xls: unknown cell type: %v", data.GetType())
	}
}

func (x XlsCell) ToBool(data structure.CellData) (bool, error) {
	switch x.Type(data) {
	case XlsCellBlank:
		return false, ErrEmptyCell
	case XlsCellBoolOrErr, XlsCellString:
		s := strings.ToUpper(data.GetString())
		switch s {
		case "TRUE":
			return true, nil
		case "FALSE":
			return false, nil
		default:
			return false, fmt.Errorf("excel/xls: string to bool %w", ErrParseError)
		}
	case XlsCellFake:
		return false, ErrNotFound
	// case XlsCellString:
	// 	return false, ErrInvalidValueType
	case XlsCellFloat:
		return false, ErrInvalidValueType
	case XlsCellInt:
		return false, ErrInvalidValueType
	case XlsCellNil:
		return false, ErrNil
	default:
		return false, fmt.Errorf("excel/xls: unknown cell type: %v", data.GetType())
	}
}

type (
	xlsRow struct {
		cols []structure.CellData
	}

	xlsSheet struct {
		rowCount int
		sheet    *xls.Sheet
	}

	xlsRowIterator struct {
		curRow int
		sheet  *xlsSheet
	}

	xlsWorkbook struct {
		workbook xls.Workbook
	}

	xlsReaderRow interface {
		GetCol(index int) (c structure.CellData, err error)
	}
)

func (x *xlsRow) ColumnCount() int {
	return len(x.cols)
}

func (x *xlsRow) GetColumn(index int) (string, error) {
	if index < 0 || index >= len(x.cols) {
		return "", ErrOutOfRange
	}
	return XlsCell{}.ToString(x.cols[index])
}

func (x *xlsRow) GetInt64Column(index int) (int64, error) {
	if index < 0 || index >= len(x.cols) {
		return 0, ErrOutOfRange
	}
	return XlsCell{}.ToInt64(x.cols[index])
}

func (x *xlsRow) GetFloat64Column(index int) (float64, error) {
	if index < 0 || index >= len(x.cols) {
		return 0, ErrOutOfRange
	}
	return XlsCell{}.ToFloat64(x.cols[index])
}

func (x *xlsRow) GetBoolColumn(index int) (bool, error) {
	if index < 0 || index >= len(x.cols) {
		return false, ErrOutOfRange
	}
	return XlsCell{}.ToBool(x.cols[index])
}

func (x *xlsRow) AllColumns() iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		for i := 0; i < len(x.cols); i++ {
			if !yield(i, x.cols[i].GetString()) {
				return
			}
		}
	}
}

func (x *xlsSheet) GetName() string {
	return x.sheet.GetName()
}

func (x *xlsSheet) RowCount() int {
	return x.rowCount
}

func (x *xlsSheet) GetRow(index int) (Row, error) {
	if index < 0 || index >= x.rowCount {
		return nil, ErrOutOfRange
	}
	row, err := x.sheet.GetRow(index)
	if err != nil {
		return nil, fmt.Errorf("excel/xls: %w", err)
	}
	cols := row.GetCols()
	return &xlsRow{cols: cols}, nil
}

func (x *xlsRowIterator) Next() bool {
	x.curRow++
	return x.curRow < x.sheet.rowCount
}

func (x *xlsRowIterator) Current() (Row, error) {
	if x.curRow < 0 {
		return nil, ErrExcelNotInitialized
	}
	if x.curRow >= x.sheet.rowCount {
		return nil, ErrEof
	}
	return x.sheet.GetRow(x.curRow)
}

func (x *xlsRowIterator) Close() error { return nil }

func (x *xlsWorkbook) SheetCount() int {
	return x.workbook.GetNumberSheets()
}

func (x *xlsWorkbook) getSheet(index int) (*xlsSheet, error) {
	sheet, err := x.workbook.GetSheet(index)
	if err != nil {
		return nil, fmt.Errorf("excel/xls: %w", err)
	}
	rowCount := 0
	if sheet != nil {
		rowCount = sheet.GetNumberRows()
	}
	return &xlsSheet{sheet: sheet, rowCount: rowCount}, nil
}

func (x *xlsWorkbook) GetSheet(index int) (Sheet, error) {
	return x.getSheet(index)
}

func (x *xlsWorkbook) IterateSheet(index int) (RowIterator, error) {
	sheet, err := x.getSheet(index)
	if err != nil {
		return nil, err
	}
	return &xlsRowIterator{curRow: -1, sheet: sheet}, nil
}

func (x *xlsWorkbook) Close() error {
	return nil
}

func NewXlsWorkbook(filePath string) (Workbook, error) {
	workbook, err := xls.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("excel/xls: %w", err)
	}
	return &xlsWorkbook{workbook: workbook}, nil
}

func NewXlsWorkbookByReadSeeker(reader io.ReadSeeker) (Workbook, error) {
	wb, err := xls.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("excel/xls: %w", err)
	}
	return &xlsWorkbook{workbook: wb}, nil
}
