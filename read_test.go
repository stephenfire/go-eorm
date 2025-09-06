package eorm

import "testing"

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
	wb, err := NewXlsWorkbook("example1.xls")
	if err != nil {
		t.Fatal(err)
	}
	rangeTest(t, wb)
}

func TestXlsxFile(t *testing.T) {
	wb, err := NewXlsxWorkbook("example2.xlsx")
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
	wb, err := NewXlsWorkbook("example1.xls")
	if err != nil {
		t.Fatal(err)
	}
	iterateTest(t, wb)
}

func TestIterXlsx(t *testing.T) {
	wb, err := NewXlsxWorkbook("example2.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	iterateTest(t, wb)
}
