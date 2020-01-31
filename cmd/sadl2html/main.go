package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/boynton/sadl"
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("usage: sadl2html model.sadl")
		os.Exit(1)
	}
	path := args[0]
	model, err := sadl.ParseFile(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	for _, hdef := range model.Http {
		snippet, err := generateHttpTrace(model, hdef)
		if err != nil {
			fmt.Println("*** Error:", err)
			os.Exit(1)
		}
		fmt.Println(snippet)
	}
}

/*func findExample(examples []*sadl.ExampleDef, typeName string) string {
	for _, ex := range examples {
		if ex.Target == typeName {
			switch v := ex.Example.(type) {
			case *string:
				return *v
			case map[string]interface{}:
				return sadl.Pretty(v)
			default:
				fmt.Printf("WHOOPS not a string: %v\n", v)
				return sadl.Pretty(ex.Example)
			}
		}
	}
	return ""
}
*/

func stringExample(ex interface{}) string {
	if s, ok := ex.(*string); ok {
		return *s
	}
	return ""
}

//to do: generate example error responses, also. Ideally, they would have requests with matching Name attributes
func generateHttpTrace(model *sadl.Model, hdef *sadl.HttpDef) (string, error) {
	examples := model.Examples
	reqType := sadl.Capitalize(hdef.Name) + "Request"
	resType := sadl.Capitalize(hdef.Name) + "Response"
	var reqExample, resExample map[string]interface{}
	//exception examples?
	for _, ex := range examples {
		if ex.Target == reqType {
			reqExample, _ = ex.Example.(map[string]interface{})
		} else if ex.Target == resType {
			resExample, _ = ex.Example.(map[string]interface{})
		}
	}
	body := ""
	if reqExample != nil {
		body = "<h2>" + hdef.Name + " request example</h2>\n"
		
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
		headers = headers + "Accept: application/json\n"
		s := method + " " + path + " HTTP/1.1\n" + headers + "\n" + bodyExample
		body = body + "<pre>\n" + s + "</pre>\n"
		
		if resExample != nil {
			body = body + "<h2>" + hdef.Name + " response example</h2>\n"
			
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
			body = body + "<pre>\n" + s + "</pre>\n"
		}
	}
	
	return "<html><head><title>HTTP Trace Examples</title></head>\n<body>\n" + body + "\n</body>\n</html>\n", nil
}

func dateHeader() string {
	t := time.Now()
	return t.Format("Mon, 2 Jan 2006 15:04:05 GMT")
}

var template = `<html>
  <head><title>HTTP Trace</title></head>
  <body>
    <h2>Example</h2>
    <h3>Request</h3>
    <pre>
      GET /people/bf938428-f04c-11e9-a280-8c8590216cf9
      Authorization: Bearer <token>
      Accept: application/json
      
    </pre>

    <h3>Response</h3>
    <pre>
      HTTP/1.1 200 OK
      Content-Length: 404
      Content-Type: application/json; charset=utf-8
      Date: Wed, 16 Oct 2019 13:46:37 GMT      
      
      {
          "id": "bf938428-f04c-11e9-a280-8c8590216cf9",
          "name": "Lee Boynton",
          "email": "lee@boynton.com"
      }
    </pre>
  </body>
</html>
`
