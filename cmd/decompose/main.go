//go:build !test

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/s0rg/decompose/internal/builder"
	"github.com/s0rg/decompose/internal/client"
	"github.com/s0rg/decompose/internal/graph"
)

const (
	appName       = "Decompose"
	appSite       = "https://github.com/s0rg/decompose"
	defaultProto  = "all"
	defaultFormat = "dot"
	defaultOutput = "-"
)

// build-time values.
var (
	GitTag    string
	GitHash   string
	BuildDate string
)

var (
	fSilent, fVersion bool
	fHelp, fLocal     bool
	fProto, fFormat   string
	fOut, fFollow     string
	fMeta             string
	fLoad             []string

	ErrUnknown = errors.New("unknown")
)

func version() string {
	return fmt.Sprintf("%s %s-%s build at: %s with %s site: %s",
		appName,
		GitTag,
		GitHash,
		BuildDate,
		runtime.Version(),
		appSite,
	)
}

func usage() {
	var sb strings.Builder

	fmt.Fprintf(&sb, "%s - reverse-engineering tool for docker environments, usage:\n\n", appName)
	fmt.Fprintf(&sb, "%s [flags]\n\n", filepath.Base(os.Args[0]))
	fmt.Fprint(&sb, "possible flags with default values:\n\n")

	_, _ = os.Stderr.WriteString(sb.String())

	flag.PrintDefaults()
}

func setupFlags() {
	flag.BoolVar(&fSilent, "silent", false, "suppress progress messages in stderr")
	flag.BoolVar(&fVersion, "version", false, "show version")
	flag.BoolVar(&fHelp, "help", false, "show this help")
	flag.BoolVar(&fLocal, "local", false, "skip external hosts")

	flag.StringVar(&fProto, "proto", defaultProto, "protocol to scan: tcp, udp or all")
	flag.StringVar(&fFollow, "follow", "", "follow only this container by name")
	flag.StringVar(&fFormat, "format", defaultFormat, "output format: json, dot or sdsl for structurizr dsl")
	flag.StringVar(&fOut, "out", defaultOutput, "output: filename or \"-\" for stdout")
	flag.StringVar(&fMeta, "meta", "", "filename with json metadata for enrichment")

	flag.Func("load", "load json stream, can be used multiple times", func(v string) error {
		fLoad = append(fLoad, v)

		return nil
	})

	flag.Usage = usage
}

func writeOut(name string, writer func(io.Writer)) error {
	var out io.Writer = os.Stdout

	if name != defaultOutput {
		fd, err := os.Create(name)
		if err != nil {
			return fmt.Errorf("create '%s': %w", name, err)
		}

		defer fd.Close()

		out = fd
	}

	writer(out)

	return nil
}

func prepareConfig(
	blder graph.Builder,
	fproto, fextra, ffollow string,
	flocal bool,
) (cfg *graph.Config, err error) {
	proto, ok := graph.ParseNetProto(fproto)
	if !ok {
		return nil, fmt.Errorf("%w protocol: %s", ErrUnknown, fproto)
	}

	extra := graph.NewMetaLoader()

	if fextra != "" {
		fd, err := os.Open(fextra)
		if err != nil {
			return nil, fmt.Errorf("extra open '%s': %w", fextra, err)
		}

		err = extra.FromReader(fd)
		fd.Close()

		if err != nil {
			return nil, fmt.Errorf("extra load '%s': %w", fextra, err)
		}
	}

	cfg = &graph.Config{
		Builder:   blder,
		Enricher:  extra,
		Proto:     proto,
		Follow:    ffollow,
		OnlyLocal: flocal,
	}

	return cfg, nil
}

func run() error {
	bldr, ok := builder.Create(fFormat)
	if !ok {
		return fmt.Errorf("%w format: %s", ErrUnknown, fFormat)
	}

	cfg, err := prepareConfig(bldr, fProto, fMeta, fFollow, fLocal)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	var act string

	if len(fLoad) > 0 {
		act = "load"
		err = doLoad(cfg, fLoad)
	} else {
		act = "build"
		err = doBuild(cfg)
	}

	if err != nil {
		return fmt.Errorf("%s: %w", act, err)
	}

	if err = writeOut(strings.TrimSpace(fOut), bldr.Write); err != nil {
		return fmt.Errorf("output: %w", err)
	}

	return nil
}

func doLoad(
	cfg *graph.Config,
	files []string,
) error {
	ldr := graph.NewLoader(cfg)

	for _, fn := range files {
		fd, err := os.Open(fn)
		if err != nil {
			return fmt.Errorf("open %s: %w", fn, err)
		}

		err = ldr.LoadStream(fd)
		fd.Close()

		if err != nil {
			return fmt.Errorf("load %s: %w", fn, err)
		}
	}

	if err := ldr.Build(); err != nil {
		return fmt.Errorf("build: %w", err)
	}

	return nil
}

func doBuild(
	cfg *graph.Config,
) error {
	cli, err := client.NewDocker()
	if err != nil {
		return fmt.Errorf("docker: %w", err)
	}

	defer cli.Close()

	log.Println("Starting with method:", cli.Kind())

	if err = graph.Build(cfg, cli); err != nil {
		return fmt.Errorf("graph: %w", err)
	}

	return nil
}

func main() {
	setupFlags()

	flag.Parse()

	if fVersion {
		fmt.Println(version())

		return
	}

	if fHelp {
		usage()

		return
	}

	if fSilent {
		log.SetOutput(io.Discard)
	}

	if err := run(); err != nil {
		log.SetOutput(os.Stderr)
		log.Fatal(err)
	}
}
