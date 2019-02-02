package sadl

import(
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/boynton/sadl/scanner"
)

var Verbose bool

func debug(args ...interface{}) {
	if Verbose {
		max := len(args) - 1
		for i := 0; i < max; i++ {
			fmt.Print(str(args[i]))
		}
		fmt.Println(str(args[max]))
	}
}

func str(arg interface{}) string {
	return fmt.Sprintf("%v", arg)
}

func BaseFileName(path string) string {
	fname := FileName(path)
	n := strings.LastIndex(fname, ".")
	if n < 1 {
		return fname
	}
	return fname[:n]
}

func FileName(path string) string {
	return filepath.Base(path)
}

func FileDir(path string) string {
	return filepath.Dir(path)
}

func Pretty(obj interface{}) string {
	j, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		return fmt.Sprint(obj)
	}
	return string(j)
}


const BLACK = "\033[0;0m"
const RED = "\033[0;31m"
const YELLOW = "\033[0;33m"
const BLUE = "\033[94m"
const GREEN = "\033[92m"

func formattedAnnotation(filename string, source string, prefix string, msg string, tok *scanner.Token, color string, contextSize int) string {
	highlight := color + "\033[1m"
	restore := BLACK + "\033[0m"
	if source != "" && contextSize >= 0 {
		lines := strings.Split(source, "\n")
		line := tok.Line - 1
		begin := max(0, line-contextSize)
		end := min(len(lines), line+contextSize+1)
		context := lines[begin:end]
		tmp := ""
		for i, l := range context {
			if i+begin == line {
				toklen := len(tok.Text)
				if toklen > 0 {
					if tok.Type == scanner.STRING {
						toklen = len(fmt.Sprintf("%q", tok.Text))
					} else if tok.Type == scanner.LINE_COMMENT {
						toklen = toklen + 2
					}
					left := ""
					mid := l
					right := ""
					if tok.Start > 0 && toklen > 1 {
						left = l[:tok.Start-1]
						mid = l[tok.Start-1:tok.Start-1+toklen]
						right = l[tok.Start-1+toklen:]
					}
					tmp += fmt.Sprintf("%3d\t%v", i+begin+1, left)
					tmp += fmt.Sprintf("%s%v%s", highlight, mid, restore)
					tmp += fmt.Sprintf("%v\n", right)
				} else {
					tmp += fmt.Sprintf("%3d\t%v\n", i+begin+1, l)
				}
			} else {
				tmp += fmt.Sprintf("%3d\t%v\n", i+begin+1, l)
			}
		}
		if tok != nil {
			if filename != "" {
				return fmt.Sprintf("%s%s:%d:%d: %s%s%s\n%s", prefix, path.Base(filename), tok.Line, tok.Start, highlight, msg, restore, tmp)
			}
			return fmt.Sprintf("%s%s%s%s\n%s", prefix, highlight, msg, restore, tmp)
		} else {
			return fmt.Sprintf("%s%d:%d: %s", prefix, tok.Line, tok.Start, msg)
		}
	}
	return fmt.Sprintf("%s: %s", prefix, msg)
}

func max(n1 int, n2 int) int {
	if n1 > n2 {
		return n1
	}
	return n2
}

func min(n1 int, n2 int) int {
	if n1 < n2 {
		return n1
	}
	return n2
}

