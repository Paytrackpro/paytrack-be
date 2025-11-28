package webserver

import (
	"net/http"

	"github.com/Paytrackpro/paytrack-be/utils"
)

type apiSettings struct {
	*WebServer
}

// getSettings handles GET /api/settings
func (a *apiSettings) getSettings(w http.ResponseWriter, r *http.Request) {
	response := a.service.GetSettings()
	utils.ResponseOK(w, response)
}
