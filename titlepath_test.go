package eorm

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stephenfire/go-tools"
)

func TestNameEncode(t *testing.T) {
	tests := []struct {
		in, out string
	}{
		{in: "反引号`测试", out: "反引号%60测试"},
		{in: "空 格", out: "空%20格"},
		{in: "斜杠/", out: "斜杠%2F"},
		{in: "双引号\"测试", out: "双引号%22测试"},
		{in: "反斜杠\\", out: "反斜杠%5C"},
		{in: "序号", out: "序号"},
	}

	for _, test := range tests {
		got := TitleEscape(test.in)
		if got != test.out {
			t.Fatalf("TitleEscape(%q) = %q, want %q", test.in, got, test.out)
		}
		un, err := TitleUnescape(got)
		if err != nil {
			t.Fatalf("TitleUnescape(%q): %v", got, err)
		}
		if un != test.in {
			t.Fatalf("TitleUnescape(%q) = %q, want %q", got, un, test.in)
		}
		fmt.Printf("[%s] => [%s] <= [%s]\n", test.in, got, un)
	}
	t.Log("TestNameEncode OK")
}

func TestMatchTitlePath(t *testing.T) {
	tests := []struct {
		path string
		val  int
	}{
		{path: "序号//", val: 0},
		{path: "名称//", val: 1},
		{path: "第一级/反引号%60测试/空%20格", val: 2},
		{path: "第一级/反引号%60测试/斜杠%2F", val: 3},
		{path: "第一级/双引号%22测试/反斜杠%5C", val: 4},
		{path: "第一级/双引号%22测试/第三级", val: 5},
		{path: "第一级/没有第三级/第三级", val: 6},
		{path: "第一级/最后一列/第三级", val: 7},
	}
	pt := new(PathTree[int])
	for _, test := range tests {
		err := pt.Put(test.val, MustTitlePath(test.path))
		if err != nil {
			t.Errorf("Put(%q): %v", test.path, err)
		}
	}

	wb, err := NewXlsWorkbook(filepath.Join("testdata", "title.xls"))
	if err != nil {
		t.Fatalf("NewXlsWorkbook: %v", err)
	}
	sheet, err := wb.GetSheet(0)
	if err != nil {
		t.Fatalf("GetSheet: %v", err)
	}
	m, err := MatchTitlePath(pt, sheet, &Params{})
	if err != nil {
		t.Fatalf("MatchTitlePath: %v", err)
	}
	t.Log("MatchTitlePath:", m)
}

func TestTitleEscape(t *testing.T) {
	bs := []byte{'%', '\'', ',', '"', '/', '\\', '\n', '\r', '\t', '`', ' '}
	ss := tools.TsToSs(func(t byte) (string, bool) {
		return string([]byte{t}), true
	}, bs...)
	tp := TitlePath(ss)
	t.Logf("%s", tp.Encode())
}
