package eorm

import "reflect"

type EORM[T any] struct {
	wb      Workbook
	objType reflect.Type
	params  *Params
}

func NewEORM(filePath string, objType reflect.Type, opts ...Option) (Workbook, error) {
	return nil, nil
}
