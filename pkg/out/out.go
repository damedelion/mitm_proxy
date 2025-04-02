package out

import (
	"fmt"
	"net/http"
)

func Request(r *http.Request) {
	fmt.Println(r.Method, r.URL, r.Proto)

	for name, values := range r.Header {
		for _, value := range values {
			fmt.Println(name, value)
		}
	}
}
