package eorm

import (
	"errors"
	"fmt"
	"strings"
)

type TitlePaths []TitlePath

func BuildTitlePaths(sheet Sheet, depth int, opts ...Option) (TitlePaths, error) {
	params := NewParams(opts...)

	if depth < 1 {
		return nil, errors.New("eorm: depth must be greater than 0")
	}
	minRows := params.MinRows(depth)
	if sheet == nil || sheet.RowCount() < minRows {
		return nil, errors.New("eorm: sheet row count must be greater than depth")
	}

	var columns TitlePaths
	appendCell := func(idx int, val string, emptyAsMerged bool) {
		for idx >= len(columns) {
			// 当前row列数多于之前列数
			if len(columns) > 0 {
				path := columns[len(columns)-1].Truncate(1)
				columns = append(columns, path)
			} else {
				columns = append(columns, TitlePath(nil))
			}
		}
		if val == "" && idx > 0 && emptyAsMerged {
			// 空被认为是合并单元格
			val = columns[idx-1].Last()
		}
		columns[idx] = append(columns[idx], val)
	}

	for i := params.TitleStartRow; i < minRows; i++ {
		emptyAsMerged := i != minRows-1 || !params.GenLastRowNoMerged
		row, err := sheet.GetRow(i)
		if err != nil {
			return nil, fmt.Errorf("eorm: get row %d: %w", i, err)
		}
		colCount := row.ColumnCount()
		j := 0
		for ; j < colCount; j++ {
			var val string
			if i != 0 || !params.GenWildcardForFirstRow {
				val, err = row.GetColumn(j)
				if err != nil && !errors.Is(err, ErrEmptyCell) {
					return nil, fmt.Errorf("eorm: get column %d: %w", j, err)
				}
				if params.TrimSpace {
					val = strings.TrimSpace(val)
				}
			}
			appendCell(j, val, emptyAsMerged)
		}
		// 当前row列数少于之前列数
		for ; j < len(columns); j++ {
			appendCell(j, "", emptyAsMerged)
		}
	}
	return columns, nil
}

func (tps TitlePaths) Info() string {
	sb := strings.Builder{}
	for i, tp := range tps {
		if i > 0 {
			sb.WriteString(fmt.Sprintln())
		}
		sb.WriteString(fmt.Sprintf("%4d: [%s]", i+1, tp))
	}
	return sb.String()
}
