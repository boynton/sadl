package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/openapi"
)

const DEFAULT_SWAGGER_VERSION = "v3.43.0"

func serveSwaggerUi(model *sadl.Model, conf *sadl.Data) error {
	gen := openapi.NewGenerator(model, conf)
	doc, err := gen.ExportToOAS3()
	if err != nil {
		return err
	}
	docPretty := sadl.Pretty(doc)
	docContent := bytes.NewReader([]byte(docPretty))
	modTime := time.Now()
	apiName := model.Name + ".json"
	fmt.Println("show swagger-ui for:", apiName, "at http://localhost:8080/")

	_ = exec.Command("open", "http://localhost:8080/").Run()

	endpoint := ":8080"
	path, err := cacheSwaggerDist()
	if err != nil {
		log.Fatalf("Cannot get swagger dist: %v\n", err)
	}
	z, err := zip.OpenReader(path)
	if err != nil {
		log.Fatalf("Cannot read zip file: %v\n", err)
	}
	prefix := "swagger-ui-3.43.0/dist/"
	for _, f := range z.File {
		tmp := strings.Split(f.Name, "/")
		if len(tmp) >= 2 && tmp[1] == "dist" {
			prefix = strings.Join(tmp[:2], "/")
			break
		}
	}
	prefixLen := len(prefix)
	files := make(map[string]*zip.File, 0)
	for _, f := range z.File {
		if strings.HasPrefix(f.Name, prefix) {
			files[f.Name[prefixLen:]] = f
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		} else if strings.HasSuffix(path, ".json") {
			http.ServeContent(w, r, path, modTime, docContent)
			return
		}
		if f, ok := files[path]; ok {
			rc, err := f.Open()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			data, err := ioutil.ReadAll(rc)
			if path == "/index.html" {
				fmt.Println("index file detected, rewrite the default URL to", apiName)
				data = []byte(strings.Replace(string(data), "https://petstore.swagger.io/v2/swagger.json", apiName, -1))
			}
			rc.Close()
			http.ServeContent(w, r, path, f.Modified, bytes.NewReader(data))
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Not Found: %q\n", path)
		}
	})
	return http.ListenAndServe(endpoint, nil)
}

func cacheSwaggerDist() (string, error) {
	version := os.Getenv("SWAGGER_RELEASE")
	if version == "" {
		version = DEFAULT_SWAGGER_VERSION
	}
	dir := os.Getenv("DOWNLOAD_DIRECTORY")
	if dir == "" {
		dir = os.Getenv("HOME") + "/Downloads"
	}
	path := dir + "/" + version + ".zip"
	if fileExists(path) {
		return path, nil
	}
	resp, err := http.Get(swaggerUrl(version))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(path, body, 0644)
	return path, err
}

func swaggerUrl(version string) string {
	return "https://github.com/swagger-api/swagger-ui/archive/" + version + ".zip"
}
