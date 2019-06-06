package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/golang"
)

func main() {
	pOutdir := flag.String("dir", "", "output directory for generated source")
	pPackage := flag.String("package", "", "Go package for generated source")
	pRuntime := flag.Bool("runtime", true, "Use SADL runtime library for base types. If false, they are generated in the target package")
	pServer := flag.Bool("server", false, "generate server code")
	flag.Parse()
	argv := flag.Args()
	argc := len(argv)
	if argc == 0 {
		fmt.Fprintf(os.Stderr, "usage: sadl2go -dir outdir -package go_package_name -runtime some_model.sadl\n")
		os.Exit(1)
	}
	if pServer != nil {
		fmt.Println("[Warning: -server NYI]")
	}
	model, err := sadl.ParseFile(argv[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "*** %v\n", err)
		os.Exit(1)
	}
	name := filepath.Base(argv[0])
	n := strings.LastIndex(name, ".")
	if n > 0 {
		name = name[:n]
	}
	gen := golang.NewGenerator(model, name, *pOutdir, *pPackage, *pRuntime)
	for _, td := range model.Types {
		gen.EmitType(td)
	}
	if gen.NeedsDecimalType() {
		gen.EmitDecimalType()
	}
	gen.EmitJsonUtil()
	gen.Finish()
}
