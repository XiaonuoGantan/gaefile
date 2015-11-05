package gaeserver

import (
	"net/http"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"

	upload "gaefile/gaeserver/upload"
)

func init() {
	var router = mux.NewRouter().StrictSlash(true)
	var n = negroni.Classic()
	n.UseHandler(router)
	upload.UploadInit(router)
	http.Handle("/", n)
}
