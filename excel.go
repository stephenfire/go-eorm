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
		IterateSheet(index int) (RowIterator, error)
		Close() error
	}

	RowIterator interface {
		Next() bool
		Current() (Row, error)
		Close() error
	}
)

var (
	ErrNotFound         = errors.New("excel: not found")
	ErrOutOfRange       = errors.New("excel: out of range")
	ErrNil              = errors.New("excel: nil")
	ErrInvalidValueType = errors.New("excel: invalid value type")
	// ErrInvalidCellValue #NULL!, #DIV/0!, #VALUE!, #REF!, #NAME?, #NUM!!, #N/A
	ErrInvalidCellValue = errors.New("excel: invalid cell value")
	ErrNotInitialized   = errors.New("excel: not initialized")
	ErrEof              = errors.New("excel: eof")
)
