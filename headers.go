package mongosm

import (
	"fmt"
	"io"
	"net/http"
)

func DisplayTrace(w io.Writer, r *http.Request) {
	fmt.Fprintf(w, "Trace is: %s\n", r.Header.Get("traceparent"))
}
