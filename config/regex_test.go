package config

import (
	"testing"
	"regexp"
	"os"
	"fmt"
)

func TestRegex(t *testing.T) {
	str := `14:46:49.671 [http-nio-8000-exec-6] ERROR o.a.c.c.C.[.[.[.[dispatcherServlet].log:182 - Servlet.service() for servlet [dispatcherServlet] in context with path [] threw exception [Request processing failed; nested exception is java.lang.IllegalArgumentException: Can not find ImageCaptcha, rid:1234567890123456] with root cause
`
	r := regexp.MustCompile("ERROR")
	if !r.MatchString(str) {
		t.Fail()
	}

	if _, err := os.Stat("testfile"); err != nil {
	} else {
		fmt.Print("no error")
		t.Fail()
	}

}
