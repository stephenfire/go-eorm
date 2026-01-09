package eorm

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stephenfire/go-common/math"
)

// WriteTestObj 用于写入测试的结构体
type WriteTestObj struct {
	ID     int64    `eorm:"序号//,required"`
	Name   string   `eorm:"名称//"`
	Email  string   `eorm:"第一级/邮箱/地址"`
	Age    int64    `eorm:"第一级/年龄/数值"`
	Active bool     `eorm:"状态//"`
	Score  *big.Int `eorm:"分数//"`
}

// SetScore 自定义setter方法
func (w *WriteTestObj) SetScore(score int64) {
	w.Score = big.NewInt(score)
}

func (w *WriteTestObj) GetScore() int64 {
	return w.Score.Int64()
}

// Equals 比较两个WriteTestObj是否相等
func (w *WriteTestObj) Equals(o *WriteTestObj) bool {
	if w == o {
		return true
	}
	if w == nil || o == nil {
		return false
	}
	return w.ID == o.ID && w.Name == o.Name && w.Email == o.Email &&
		w.Age == o.Age && w.Active == o.Active &&
		math.CompareBigInt(w.Score, o.Score) == 0
}

func (w *WriteTestObj) String() string {
	if w == nil {
		return "<nil>"
	}
	return fmt.Sprintf("{ID:%d Name:%s Email:%s Age:%d Active:%t Score:%s}",
		w.ID, w.Name, w.Email, w.Age, w.Active, math.BigIntForPrint(w.Score))
}

func runWriterTest(t *testing.T, runner string,
	run func(tmpFilePath string, wb Workbook, sheetWriter SheetWriter, writer *EORMWriter[WriteTestObj])) {
	// 使用testdata中的现有文件作为模板
	srcPath := filepath.Join("testdata", "write_template.xlsx")

	// 创建临时副本
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("test_%s_*.xlsx", runner))
	if err != nil {
		t.Fatal(err)
	}
	tmpFilePath := tmpFile.Name()
	tmpFile.Close()

	// 复制文件
	data, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpFilePath, data, 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFilePath)

	// 创建Workbook
	wb, err := NewWorkbook(tmpFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer wb.Close()

	// 获取SheetWriter
	sheetWriter, err := wb.GetSheetWriter(0)
	if err != nil {
		t.Fatal(err)
	}

	// 测试NewWriter
	writer, err := NewWriter[WriteTestObj](sheetWriter, reflect.TypeOf(WriteTestObj{}), WithTitleStartRow(2))
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	run(tmpFilePath, wb, sheetWriter, writer)
}

// TestNewWriter 测试NewWriter函数
func TestNewWriter(t *testing.T) {
	runWriterTest(t, "writer",
		func(tmpFilePath string, wb Workbook, sheetWriter SheetWriter, writer *EORMWriter[WriteTestObj]) {
			if writer == nil {
				t.Fatal("writer is nil")
			}

			if writer.w != sheetWriter {
				t.Error("writer.w does not match sheetWriter")
			}

			if writer.rowMapper == nil {
				t.Error("rowMapper is nil")
			}

			// 2个空行+3层表头
			if writer.curRowIndex != 4 {
				t.Errorf("expected curRowIndex 4, got %d", writer.curRowIndex)
			}
		})
}

// TestWriterAppend 测试Append方法
func TestWriterAppend(t *testing.T) {
	runWriterTest(t, "append",
		func(tmpFilePath string, wb Workbook, sheetWriter SheetWriter, writer *EORMWriter[WriteTestObj]) {
			// 测试数据
			testObjs := []*WriteTestObj{
				{
					ID:     1,
					Name:   "Alice",
					Email:  "alice@example.com",
					Age:    30,
					Active: true,
					Score:  big.NewInt(95),
				},
				{
					ID:     2,
					Name:   "Bob",
					Email:  "bob@example.com",
					Age:    25,
					Active: false,
					Score:  big.NewInt(85),
				},
				{
					ID:     3,
					Name:   "Charlie",
					Email:  "charlie@example.com",
					Age:    35,
					Active: true,
					Score:  big.NewInt(90),
				},
			}

			// 测试Append
			n, err := writer.Append(testObjs...)
			if err != nil {
				t.Fatalf("Append failed: %v", err)
			}

			if n != len(testObjs) {
				t.Errorf("expected wrote %d objects, got %d", len(testObjs), n)
			}

			if err = wb.Save(); err != nil {
				t.Fatalf("save failed: %v", err)
			}
			// 保存文件
			wb.Close()

			// 重新打开文件验证写入的内容
			wb2, err := NewWorkbook(tmpFilePath)
			if err != nil {
				t.Fatal(err)
			}
			defer wb2.Close()

			sheet, err := wb2.GetSheet(0)
			if err != nil {
				t.Fatal(err)
			}

			// 创建Reader验证数据
			reader, err := NewReader[WriteTestObj](sheet, reflect.TypeOf(WriteTestObj{}), WithTitleStartRow(2))
			if err != nil {
				t.Fatalf("NewReader failed: %v", err)
			}

			// 验证读取的数据
			i := 0
			for reader.Next() {
				if i >= len(testObjs) {
					t.Fatalf("expected %d rows, got more", len(testObjs))
				}

				rowObj, err := reader.Current()
				if err != nil {
					t.Fatal(err)
				}

				if !testObjs[i].Equals(rowObj) {
					t.Errorf("row %d mismatch: expected %v, got %v", i, testObjs[i], rowObj)
				}
				i++
			}

			if i != len(testObjs) {
				t.Errorf("expected %d rows, got %d", len(testObjs), i)
			}
		})

}

// TestWriterAppendWithNil 测试Append方法处理nil对象
func TestWriterAppendWithNil(t *testing.T) {
	runWriterTest(t, "append_nil",
		func(tmpFilePath string, wb Workbook, sheetWriter SheetWriter, writer *EORMWriter[WriteTestObj]) {
			// 混合nil和非nil对象
			testObjs := []*WriteTestObj{
				{
					ID:     1,
					Name:   "Alice",
					Email:  "alice@example.com",
					Age:    30,
					Active: true,
					Score:  big.NewInt(95),
				},
				nil,
				{
					ID:     2,
					Name:   "Bob",
					Email:  "bob@example.com",
					Age:    25,
					Active: false,
					Score:  big.NewInt(85),
				},
				nil,
				nil,
			}

			n, err := writer.Append(testObjs...)
			if err != nil {
				t.Fatalf("Append failed: %v", err)
			}

			// 应该只写入2个非nil对象
			expectedWrote := 2
			if n != expectedWrote {
				t.Errorf("expected wrote %d objects, got %d", expectedWrote, n)
			}

			// 验证curRowIndex
			// 初始curRowIndex=4，写入2个对象后应该是6
			if writer.curRowIndex != 6 {
				t.Errorf("expected curRowIndex 6, got %d", writer.curRowIndex)
			}
		})
}

// TestWriterSkipRows 测试SkipRows方法
func TestWriterSkipRows(t *testing.T) {
	runWriterTest(t, "skip_rows",
		func(tmpFilePath string, wb Workbook, sheetWriter SheetWriter, writer *EORMWriter[WriteTestObj]) {
			// 记录初始curRowIndex
			initialIndex := writer.curRowIndex

			// 跳过3行
			writer.SkipRows(3)
			if writer.curRowIndex != initialIndex+3 {
				t.Errorf("after SkipRows(3): expected curRowIndex %d, got %d", initialIndex+3, writer.curRowIndex)
			}

			// 写入一个对象
			obj := &WriteTestObj{
				ID:     1,
				Name:   "Test",
				Email:  "test@example.com",
				Age:    20,
				Active: true,
				Score:  big.NewInt(100),
			}

			n, err := writer.Append(obj)
			if err != nil {
				t.Fatalf("Append after SkipRows failed: %v", err)
			}

			if n != 1 {
				t.Errorf("expected wrote 1 object, got %d", n)
			}

			// 跳过2行
			writer.SkipRows(2)
			if writer.curRowIndex != initialIndex+3+1+2 {
				t.Errorf("after SkipRows(2): expected curRowIndex %d, got %d", initialIndex+3+1+2, writer.curRowIndex)
			}
		})
}

// TestWriterAppendEmpty 测试Append空参数
func TestWriterAppendEmpty(t *testing.T) {
	runWriterTest(t, "append_empty",
		func(tmpFilePath string, wb Workbook, sheetWriter SheetWriter, writer *EORMWriter[WriteTestObj]) {
			// curRowIndex应该不变
			initialIndex := writer.curRowIndex
			// 测试空参数
			n, err := writer.Append()
			if err != nil {
				t.Fatalf("Append with empty args failed: %v", err)
			}

			if n != 0 {
				t.Errorf("expected wrote 0 objects, got %d", n)
			}

			if writer.curRowIndex != initialIndex {
				t.Errorf("curRowIndex changed from %d to %d after empty Append", initialIndex, writer.curRowIndex)
			}
		})
}

// TestWriterWithExistingData 测试在已有数据的sheet上写入
func TestWriterWithExistingData(t *testing.T) {
	runWriterTest(t, "existing",
		func(tmpFilePath string, wb Workbook, sheetWriter SheetWriter, writer *EORMWriter[WriteTestObj]) {
			// 获取当前行号
			currentRow := writer.curRowIndex

			// 写入新数据
			obj := &WriteTestObj{
				ID:     100,
				Name:   "NewData",
				Email:  "new@example.com",
				Age:    40,
				Active: true,
				Score:  big.NewInt(99),
			}

			n, err := writer.Append(obj)
			if err != nil {
				t.Fatalf("Append failed: %v", err)
			}

			if n != 1 {
				t.Errorf("expected wrote 1 object, got %d", n)
			}

			// 验证curRowIndex
			if writer.curRowIndex != currentRow+1 { // +1 for new row
				t.Errorf("expected curRowIndex %d, got %d", currentRow+1, writer.curRowIndex)
			}
		})
}
