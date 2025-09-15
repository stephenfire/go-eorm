package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"sync/atomic"
	"syscall"

	"github.com/stephenfire/go-common/log"
	"github.com/stephenfire/go-eorm"
	"github.com/urfave/cli/v2"
)

var (
	fileFlag = &cli.StringFlag{
		Name:     "file",
		Usage:    "the input excel `FILE`",
		Required: true,
		Aliases:  []string{"f"},
	}

	depthFlag = &cli.IntFlag{
		Name:     "depth",
		Usage:    "specify the first `DEPTH` rows of excel as the title path",
		Required: true,
		Aliases:  []string{"d"},
	}

	firstWildcardFlag = &cli.BoolFlag{
		Name:    "wildcard-first-line",
		Usage:   "set to true to turn on `GenWildcardForFirstLayer` parameter",
		Value:   false,
		Aliases: []string{"w1"},
	}

	trimSpaceFlag = &cli.BoolFlag{
		Name:    "trim-space",
		Usage:   "set to true to turn on `TrimSpace` parameter",
		Value:   false,
		Aliases: []string{"t"},
	}

	lastLayerEmptyNotAsMergedFlag = &cli.BoolFlag{
		Name:    "last-layer-empty-not-as-merged",
		Usage:   "set to true to turn on `GenLastLayerNoMerged` parameter",
		Value:   false,
		Aliases: []string{"m"},
	}

	allFlags = []cli.Flag{
		fileFlag,
		depthFlag,
		firstWildcardFlag,
		lastLayerEmptyNotAsMergedFlag,
		trimSpaceFlag,
	}
)

func main() {
	app := &cli.App{
		Name:      "pathgener",
		Usage:     "generate title paths for an excel file",
		Version:   eorm.Version.String(),
		Copyright: eorm.Copyright,
		Flags:     allFlags,
		Action:    pathgener,
	}
	sort.Sort(cli.CommandsByName(app.Commands))
	sort.Sort(cli.FlagsByName(app.Flags))
	for _, cmd := range app.Commands {
		sort.Sort(cli.FlagsByName(cmd.Flags))
	}
	var canceled atomic.Bool
	baseCtx, cancel := context.WithCancel(context.Background())
	go func() {
		defer func() {
			if canceled.CompareAndSwap(false, true) {
				cancel()
			}
		}()
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		ss := <-sigs
		log.Warnf("GOT A SYSTEM SIGNAL[%s]\n", ss.String())
	}()
	if err := app.RunContext(baseCtx, os.Args); err != nil {
		log.Errorf("exit from main: %v", err)
		if canceled.CompareAndSwap(false, true) {
			cancel()
		}
	}
}

func pathgener(ctx *cli.Context) error {
	filename := ctx.String(fileFlag.Name)
	depth := ctx.Int(depthFlag.Name)
	wb, err := eorm.NewWorkbook(filename)
	if err != nil {
		return err
	}
	sheet, err := wb.GetSheet(0)
	if err != nil {
		return err
	}
	var opts []eorm.Option
	if ctx.Bool(firstWildcardFlag.Name) {
		opts = append(opts, eorm.WithGenWildcardForFirstLayer())
	}
	if ctx.Bool(trimSpaceFlag.Name) {
		opts = append(opts, eorm.WithTrimSpace())
	}
	if ctx.Bool(lastLayerEmptyNotAsMergedFlag.Name) {
		opts = append(opts, eorm.WithGenLastLayerNoMerged())
	}
	tps, err := eorm.BuildTitlePaths(sheet, depth, opts...)
	if err != nil {
		return err
	}
	fmt.Println(tps.Info())
	return nil
}
