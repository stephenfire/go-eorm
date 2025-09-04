module github.com/stephenfire/go-eorm

go 1.24

require github.com/shakinm/xlsReader v0.9.12

require (
	github.com/metakeule/fmtdate v1.1.2 // indirect
	golang.org/x/text v0.25.0 // indirect
)

replace (
	github.com/shakinm/xlsReader v0.9.12 => github.com/stephenfire/xlsReader v0.0.1
	github.com/xuri/excelize/v2 v2.9.1 => github.com/qax-os/excelize/v2 v2.9.1
)
