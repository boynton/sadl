package httptrace

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/boynton/sadl"
)

func Export(model *sadl.Model, conf *sadl.Data) error {
	for _, hdef := range model.Http {
		snippet, err := generateHttpTrace(model, hdef)
		if err != nil {
			fmt.Println("*** Error:", err)
			os.Exit(1)
		}
		fmt.Println(snippet)
	}
	return nil
}

func stringExample(ex interface{}) string {
	switch v := ex.(type) {
	case *string:
		return *v
	case *sadl.Decimal:
		return v.String()
	}
	return ""
}

func generateHttpTrace(model *sadl.Model, hdef *sadl.HttpDef) (string, error) {
	examples := model.Examples
	reqType := sadl.Capitalize(hdef.Name) + "Request"
	resType := sadl.Capitalize(hdef.Name) + "Response"
	namedExamples := make(map[string][]map[string]interface{}, 0)
	var reqExample, resExample map[string]interface{}

	//exception examples?
	//each named example should be a pair of req/res
	for _, ex := range examples {
		if ex.Target == reqType {
			namedExamples[ex.Name] = []map[string]interface{}{ex.Example.(map[string]interface{})}
		}
	}
	for _, ex := range examples {
		if ex.Target == resType {
			namedExamples[ex.Name] = append(namedExamples[ex.Name], ex.Example.(map[string]interface{}))
		}
	}
	body := ""
	for exName, exlist := range namedExamples {
		if len(exlist) != 2 {
			continue
		}
		reqExample = exlist[0]
		resExample = exlist[1]
		if resExample == nil {
			panic("whoops, no matching response")
		}
		body = body + "#\n# " + exName + " (action=" + hdef.Name + ")\n#\n"
		if reqExample != nil {
			body = body + "#   Request:"
			
			method := hdef.Method
			path := hdef.Path
			bodyExample := ""
			headers := ""
			
			for _, in := range hdef.Inputs {
				ex := reqExample[in.Name]
				if in.Path || in.Query != "" {
					sex := stringExample(ex)
					if in.Path {
						//inExample = urlEncode(inExample)
					}
					path = strings.Replace(path, "{" + in.Name + "}", sex, -1)
				} else if in.Header != "" {
					sex := stringExample(ex)
					headers = headers + in.Header + ": " + sex + "\n"
				} else { //body
					bodyExample = sadl.Pretty(ex)
				}
			}
			path = stripMissingOptionalQueryParams(path)
			headers = headers + "Accept: application/json\n"
			s := method + " " + path + " HTTP/1.1\n" + headers + "\n" + bodyExample
			body = body + "\n" + s + "\n"
			
			if resExample != nil {
				body = body + "#   Response:"
				
				bodyExample := ""
				headers := "Content-Type: application/json; charset=utf-8\n"
				
				for _, out := range hdef.Expected.Outputs {
					ex := resExample[out.Name]
					if out.Header != "" {
						sex := stringExample(ex)
						headers = headers + out.Header + ": " + sex + "\n"
					} else { //body
						bodyExample = sadl.Pretty(ex)
					}
				}
				headers = headers + "Date: " + dateHeader() + "\n"
				headers = fmt.Sprintf("Content-Length: %d\n", len(bodyExample)) + headers
				respMessage := fmt.Sprintf("HTTP/1.1 %d %s\n", hdef.Expected.Status, http.StatusText(int(hdef.Expected.Status)))
				s := respMessage + headers + "\n" + bodyExample
				body = body + "\n" + s + "\n"
			}
		}
	}
	return body, nil
}

func dateHeader() string {
	t := time.Now()
	return t.Format("Mon, 2 Jan 2006 15:04:05 GMT")
}

func stripMissingOptionalQueryParams(uri string) string {
	n := strings.Index(uri, "?")
	if n < 0 {
		return uri
	}
	path := uri[:n+1]
	query := uri[n+1:]
	items := strings.Split(query, "&")
	newItems := make([]string, 0)
	for _, item := range items {
		if !strings.HasSuffix(item, "=") {
			newItems = append(newItems, item)
		}
	}
	if len(newItems) == 0 {
		return path[:len(path)-1]
	}
	return path + strings.Join(newItems, "&")
}
