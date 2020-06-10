package httptrace

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/util"
)

func Export(model *sadl.Model, conf map[string]interface{}) error {
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
	if s, ok := ex.(*string); ok {
		return *s
	}
	return ""
}

//to do: generate example error responses, also. Ideally, they would have requests with matching Name attributes
func generateHttpTrace(model *sadl.Model, hdef *sadl.HttpDef) (string, error) {
	examples := model.Examples
	reqType := util.Capitalize(hdef.Name) + "Request"
	resType := util.Capitalize(hdef.Name) + "Response"
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
		body = "# " + hdef.Name + " request example\n"
		
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
				bodyExample = util.Pretty(ex)
			}
		}
		headers = headers + "Accept: application/json\n"
		s := method + " " + path + " HTTP/1.1\n" + headers + "\n" + bodyExample
		body = body + "\n" + s + "\n"
		
		if resExample != nil {
			body = body + "# " + hdef.Name + " response example\n"
			
			bodyExample := ""
			headers := "Content-Type: application/json; charset=utf-8\n"
			
			for _, out := range hdef.Expected.Outputs {
				ex := resExample[out.Name]
				if out.Header != "" {
					sex := stringExample(ex)
					headers = headers + out.Header + ": " + sex + "\n"
				} else { //body
					bodyExample = util.Pretty(ex)
				}
			}
			headers = headers + "Date: " + dateHeader() + "\n"
			headers = fmt.Sprintf("Content-Length: %d\n", len(bodyExample)) + headers
			respMessage := fmt.Sprintf("HTTP/1.1 %d %s\n", hdef.Expected.Status, http.StatusText(int(hdef.Expected.Status)))
			s := respMessage + headers + "\n" + bodyExample
			body = body + "\n" + s + "\n"
		}
	}
	
	return body, nil
}

func dateHeader() string {
	t := time.Now()
	return t.Format("Mon, 2 Jan 2006 15:04:05 GMT")
}
