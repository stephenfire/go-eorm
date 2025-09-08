package eorm

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func rangeTest(t *testing.T, wb Workbook) {
	for i := 0; i < wb.SheetCount(); i++ {
		sheet, err := wb.GetSheet(i)
		if err != nil {
			t.Fatal(err)
		}
		for j := 0; j < sheet.RowCount(); j++ {
			row, err := sheet.GetRow(j)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("Row %d-%d:", i, j)
			for k := 0; k < row.ColumnCount(); k++ {
				v, err := row.GetColumn(k)
				if err != nil {
					t.Fatal(err)
				}
				t.Logf("\tColumn %d: %s", k, v)
			}
		}
	}
}

func TestXlsFile(t *testing.T) {
	wb, err := NewXlsWorkbook(filepath.Join("testdata", "example1.xls"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = wb.Close()
	}()
	rangeTest(t, wb)
}

func TestXlsxFile(t *testing.T) {
	wb, err := NewXlsxWorkbook(filepath.Join("testdata", "example2.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	rangeTest(t, wb)
}

func iterateTest(t *testing.T, wb Workbook) {
	for i := 0; i < wb.SheetCount(); i++ {
		it, err := wb.IterateSheet(i)
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for it.Next() {
			row, err := it.Current()
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("Row %d-%d:", i, count)
			for k := 0; k < row.ColumnCount(); k++ {
				v, err := row.GetColumn(k)
				if err != nil {
					t.Fatal(err)
				}
				t.Logf("\tColumn %d: %s", k, v)
			}
			count++
		}
		_ = it.Close()
	}
}

func TestIterXls(t *testing.T) {
	wb, err := NewXlsWorkbook(filepath.Join("testdata", "example1.xls"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = wb.Close()
	}()
	iterateTest(t, wb)
}

func TestIterXlsx(t *testing.T) {
	wb, err := NewXlsxWorkbook(filepath.Join("testdata", "example2.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	iterateTest(t, wb)
}

func TestTitle(t *testing.T) {
	wb, err := NewXlsxWorkbook(filepath.Join("testdata", "title.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	rangeTest(t, wb)
}

func TestXlsTitle(t *testing.T) {
	wb, err := NewXlsWorkbook(filepath.Join("testdata", "title.xls"))
	if err != nil {
		t.Fatal(err)
	}
	rangeTest(t, wb)
}

func TestIterTitle(t *testing.T) {
	wb, err := NewXlsxWorkbook(filepath.Join("testdata", "title.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	iterateTest(t, wb)
}

func TestAllMergeCells(t *testing.T) {
	f, err := excelize.OpenFile(filepath.Join("testdata", "title.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = f.Close()
	}()
	// 获取指定工作表的合并单元格信息
	mergeCells, err := f.GetMergeCells("Sheet1")
	if err != nil {
		fmt.Println(err)
		return
	}

	// 遍历所有合并单元格
	for _, mc := range mergeCells {
		fmt.Printf("合并区域: %s - %s, 起始值: %s\n", mc.GetStartAxis(), mc.GetEndAxis(), mc.GetCellValue())
	}
}
