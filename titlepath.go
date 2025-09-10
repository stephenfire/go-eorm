package eorm

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/stephenfire/go-tools"
)

func shouldEscape(c byte) bool {
	switch c {
	case '%', '\'', '"', '/', '\\', '\n', '\r', '\t', '`', ' ':
		return true
	}
	return false
}

func ishex(c byte) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	}
	return false
}

func unhex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	default:
		panic("invalid hex character")
	}
}

const (
	upperhex  = "0123456789ABCDEF"
	separator = "/"
)

type EscapeError string

func (e EscapeError) Error() string {
	return "eorm: invalid title escape " + strconv.Quote(string(e))
}

func TitleEscape(t string) string {
	count := 0
	for i := 0; i < len(t); i++ {
		c := t[i]
		if shouldEscape(c) {
			count++
		}
	}
	if count == 0 {
		return t
	}

	var buf [64]byte
	var s []byte

	required := len(t) + 2*count
	if required <= len(buf) {
		s = buf[:required]
	} else {
		s = make([]byte, required)
	}

	j := 0
	for i := 0; i < len(t); i++ {
		c := t[i]
		if shouldEscape(c) {
			s[j] = '%'
			s[j+1] = upperhex[c>>4]
			s[j+2] = upperhex[c&0x0f]
			j += 3
		} else {
			s[j] = c
			j++
		}
	}
	return string(s)
}

func TitleUnescape(t string) (string, error) {
	n := 0
	for i := 0; i < len(t); {
		switch t[i] {
		case '%':
			n++
			if i+2 >= len(t) || !ishex(t[i+1]) || !ishex(t[i+2]) {
				t = t[i:]
				if len(t) > 3 {
					t = t[:3]
				}
				return "", EscapeError(t)
			}
			i += 3
		default:
			i++
		}
	}
	if n == 0 {
		return string(t), nil
	}
	var b bytes.Buffer
	b.Grow(len(t) - 2*n)
	for i := 0; i < len(t); i++ {
		switch t[i] {
		case '%':
			b.WriteByte(unhex(t[i+1])<<4 | unhex(t[i+2]))
			i += 2
		default:
			b.WriteByte(t[i])
		}
	}
	return b.String(), nil
}

// TitlePath 用来指定在多级（包括单级）表头中的路径，从而确定指向的列。
type TitlePath []string

func (tp TitlePath) Encode() string {
	if len(tp) == 0 {
		return ""
	}
	parts := make([]string, 0, len(tp))
	for _, name := range tp {
		parts = append(parts, TitleEscape(name))
	}
	return strings.Join(parts, separator)
}

func (tp TitlePath) Decode(namepath string) (TitlePath, error) {
	parts := strings.Split(namepath, separator)
	for i, part := range parts {
		title, err := TitleUnescape(part)
		if err != nil {
			return nil, err
		}
		parts[i] = title
	}
	return parts, nil
}

func MustTitlePath(path string) TitlePath {
	tp, err := TitlePath(nil).Decode(path)
	if err != nil {
		panic(err)
	}
	return tp
}

type (
	// TreeItem 只支持由上而下的树状结构构建的表头，每一个TreeItem都有其明确的父节点
	TreeItem[T any] interface {
		IsValue() bool
		HasValue() bool
		GetValue() T
		IsBranch() bool
		HasChild(title string) bool
		GetChild(title string) TreeItem[T]
		SetChild(title string, child TreeItem[T]) error
		ChildrenKeys() []string
		Depth() (int, error)
	}

	PathTree[T any] struct {
		depth int
		root  TreeItem[T]
	}

	branch[T any] map[string]TreeItem[T]
	value[T any]  struct {
		v *T
	}
)

var (
	ErrEmptyPath   = errors.New("eorm: empty path")
	ErrUnsupported = errors.New("eorm: unsupported")
)

func newBranch[T any]() branch[T]                                  { return make(branch[T]) }
func (b branch[T]) IsValue() bool                                  { return false }
func (b branch[T]) HasValue() bool                                 { return false }
func (b branch[T]) GetValue() (t T)                                { return t }
func (b branch[T]) IsBranch() bool                                 { return true }
func (b branch[T]) HasChild(title string) bool                     { _, ok := b[title]; return ok }
func (b branch[T]) GetChild(title string) TreeItem[T]              { return b[title] }
func (b branch[T]) SetChild(title string, child TreeItem[T]) error { b[title] = child; return nil }
func (b branch[T]) ChildrenKeys() []string                         { return slices.Collect(maps.Keys(b)) }
func (b branch[T]) Depth() (int, error) {
	if len(b) == 0 {
		return 0, errors.New("eorm: empty branch")
	}
	depth := -1
	for _, child := range b {
		if child == nil {
			return 0, errors.New("eorm: child is nil")
		}
		d, err := child.Depth()
		if err != nil {
			return 0, err
		}
		if d < 0 {
			return 0, errors.New("eorm: child depth is negative")
		}
		if depth == -1 {
			depth = 1 + d
		} else {
			if depth != d+1 {
				return 0, errors.New("eorm: child depth mismatch")
			}
		}
	}
	if depth <= 0 {
		// should not be here
		return 0, errors.New("eorm: no child")
	}
	return depth, nil
}

func (value[T]) IsValue() bool    { return true }
func (v value[T]) HasValue() bool { return v.v != nil }
func (v value[T]) GetValue() (t T) {
	if v.v != nil {
		return *v.v
	}
	return t
}
func (v value[T]) IsBranch() bool                         { return false }
func (v value[T]) HasChild(_ string) bool                 { return false }
func (v value[T]) GetChild(_ string) (t TreeItem[T])      { return t }
func (v value[T]) SetChild(_ string, _ TreeItem[T]) error { return ErrUnsupported }
func (v value[T]) ChildrenKeys() []string                 { return nil }
func (v value[T]) Depth() (int, error)                    { return 0, nil }

func (p *PathTree[T]) Depth() int { return p.depth }

func (p *PathTree[T]) Check() (depth int, err error) {
	if p == nil || p.root == nil {
		return 0, ErrEmptyPath
	}
	if !p.root.IsBranch() {
		return 0, errors.New("eorm: root not a branch")
	}
	depth, err = p.root.Depth()
	if err != nil {
		return 0, err
	}
	if depth != p.depth {
		return 0, errors.New("eorm: depth mismatch")
	}
	return depth, nil
}

func (p *PathTree[T]) Put(val T, path TitlePath) error {
	if len(path) == 0 {
		return ErrEmptyPath
	}
	if p.root == nil {
		p.depth = len(path)
	} else {
		if p.depth != len(path) {
			return fmt.Errorf("eorm: path depth not match, expecting %d, got %d", p.depth, len(path))
		}
	}

	if p.root == nil {
		p.root = newBranch[T]()
	}
	item := p.root
	for i, title := range path {
		if !item.IsBranch() {
			return errors.New("eorm: path item is not a branch")
		}
		if i == len(path)-1 {
			// last title
			if item.HasChild(title) {
				return errors.New("eorm: path item already has child")
			}
			if err := item.SetChild(title, &value[T]{v: &val}); err != nil {
				return err
			}
		} else {
			child := item.GetChild(title)
			if child == nil {
				child = newBranch[T]()
				if err := item.SetChild(title, child); err != nil {
					return err
				}
			}
			item = child
		}
	}
	return nil
}

// TitleLayer 从 PathTree 的根开始，带有传承的记录每一级根据列内容匹配的列和对应的节点
// 当 key == -1 时，表示所有列都匹配的节点。初始化时的值为 {-1: root}
type TitleLayer[T any] struct {
	m        map[int]TreeItem[T]
	maxWidth int
}

func NewTitleLayer[T any](root TreeItem[T]) *TitleLayer[T] {
	return &TitleLayer[T]{
		m:        map[int]TreeItem[T]{-1: root},
		maxWidth: 0,
	}
}

func (m *TitleLayer[T]) Size() int {
	return len(m.m)
}

func (m *TitleLayer[T]) At(index int) (TreeItem[T], bool) {
	item, ok := m.m[index]
	if ok && item != nil {
		return item, true
	}
	item, ok = m.m[-1]
	return item, ok && item != nil
}

func (m *TitleLayer[T]) NextRow(row Row) (*TitleLayer[T], error) {
	lastVal := ""
	var next tools.KMap[int, TreeItem[T]]
	putNext := func(idx int) {
		item, ok := m.At(idx)
		if !ok {
			return
		}
		child := item.GetChild(lastVal)
		if child != nil {
			next = next.Put(idx, child)
		}
	}
	colCount := row.ColumnCount()
	for i := 0; i < colCount; i++ {
		// 为了在内容为空时使用前面的值填充，所以按顺序读取所有列，一一进行匹配
		val, err := row.GetColumn(i)
		if err != nil {
			return nil, fmt.Errorf("eorm: get column %d: %w", i, err)
		}
		if val == "" {
			val = lastVal
		} else {
			lastVal = val
		}
		putNext(i)
	}
	for i := colCount; i < m.maxWidth; i++ {
		putNext(i)
	}
	return &TitleLayer[T]{m: next, maxWidth: max(colCount, m.maxWidth)}, nil
}

func (m *TitleLayer[T]) Values() (map[int]T, error) {
	var ret tools.KMap[int, T]
	for idx, item := range m.m {
		if item == nil {
			continue
		}
		if !item.IsValue() {
			return nil, fmt.Errorf("eorm: not a value at column %d", idx)
		}
		ret = ret.Put(idx, item.GetValue())
	}
	return ret, nil
}

func MatchTitlePath[T any](tree *PathTree[T], sheet Sheet) (map[int]T, error) {
	depth, err := tree.Check()
	if err != nil {
		return nil, err
	}
	rowCount := sheet.RowCount()
	if rowCount < depth {
		return nil, errors.New("eorm: row not enough")
	}
	layer := NewTitleLayer(tree.root)
	for i := 0; i < depth; i++ {
		row, err := sheet.GetRow(i)
		if err != nil {
			return nil, fmt.Errorf("eorm: get row %d: %w", i, err)
		}
		if row == nil {
			return nil, fmt.Errorf("eorm: get row %d nil", i)
		}
		layer, err = layer.NextRow(row)
		if err != nil {
			return nil, fmt.Errorf("eorm: layer next row %d: %w", i, err)
		}
	}
	if layer.Size() == 0 {
		return nil, nil
	}
	return layer.Values()
}
