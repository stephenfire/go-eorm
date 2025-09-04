package eorm

import "errors"

type (
	Row interface {
		ColumnCount() int
		GetColumn(index int) (string, error)
		GetInt64Column(index int) (int64, error)
		GetFloat64Column(index int) (float64, error)
		GetBoolColumn(index int) (bool, error)
	}

	Sheet interface {
		RowCount() int
		GetRow(index int) (Row, error)
	}

	Workbook interface {
		SheetCount() int
		GetSheet(index int) (Sheet, error)
	}
)

var (
	ErrNotFound         = errors.New("eorm: not found")
	ErrOutOfRange       = errors.New("eorm: out of range")
	ErrNil              = errors.New("eorm: nil")
	ErrInvalidValueType = errors.New("eorm: invalid value type")
	// ErrInvalidCellValue #NULL!, #DIV/0!, #VALUE!, #REF!, #NAME?, #NUM!!, #N/A
	ErrInvalidCellValue = errors.New("eorm: invalid cell value")
)
