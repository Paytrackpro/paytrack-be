package webserver

import "net/http"

type apiUser struct {
	*WebServer
}

func (a *apiUser) info(w http.ResponseWriter, r *http.Request) {

}

func (a *apiUser) update(w http.ResponseWriter, r *http.Request) {

}
