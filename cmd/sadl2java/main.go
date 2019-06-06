package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/boynton/sadl/exporters/java"
	"github.com/boynton/sadl/parse"
)

func main() {
	pDir := flag.String("dir", ".", "output directory for generated artifacts")
	pSrc := flag.String("src", "src/main/java", "output directory for generated source tree, relative to dir")
	pRez := flag.String("rez", "src/main/resources", "output directory for generated resources, relative to dir")
	pPackage := flag.String("package", "", "Java package for generated source")
	pServer := flag.Bool("server", false, "generate server code")
	pLombok := flag.Bool("lombok", false, "generate Lombok annotations")
	pGetters := flag.Bool("getters", false, "generate setters/getters instead of the default fluent style")
	pInstants := flag.Bool("instants", false, "Use java.time.Instant. By default, use generated Timestamp class")
	pPom := flag.Bool("pom", false, "Create Maven pom.xml file to build the project")
	flag.Parse()
	argv := flag.Args()
	argc := len(argv)
	if argc == 0 {
		fmt.Fprintf(os.Stderr, "usage: sadl2java -dir projdir -src relative_source_dir -rez relative_resource_dir -package java.package.name -pom -server -getters -lombok some_model.sadl\n")
		os.Exit(1)
	}
	model, err := parse.File(argv[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "*** %v\n", err)
		os.Exit(1)
	}
	gen := java.NewGenerator(model, *pDir, *pSrc, *pRez, *pPackage, *pLombok, *pGetters, *pInstants)
	for _, td := range model.Types {
		gen.CreatePojoFromDef(td)
	}
	if gen.NeedTimestamps {
		gen.CreateTimestamp()
	}
	gen.CreateJsonUtil()
	if gen.Err != nil {
		fmt.Fprintf(os.Stderr, "*** %v\n", err)
		os.Exit(1)
	}
	if *pServer {
		gen.CreateServer(*pSrc, *pRez)
	}
	if *pPom {
		domain := os.Getenv("DOMAIN")
		if domain == "" {
			domain = "my.domain"
		}
		gen.CreatePom(domain, model.Name, *pDir, *pLombok, "")
	}
	if gen.Err != nil {
		fmt.Fprintf(os.Stderr, "*** %v\n", gen.Err)
		os.Exit(1)
	}
}
