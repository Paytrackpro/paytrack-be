package webserver

import (
	"net/http"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
)

type apiProject struct {
	*WebServer
}

func (a *apiProject) createProject(w http.ResponseWriter, r *http.Request) {
	var body portal.ProjectRequest
	err := a.parseJSONAndValidate(r, &body)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}
	claims, _ := a.credentialsInfo(r)
	project, err := a.service.CreateNewProject(claims.Id, body)
	if err != nil {
		utils.Response(w, http.StatusForbidden, utils.NewError(err, utils.ErrorForbidden), nil)
	}
	utils.ResponseOK(w, project)
}

func (a *apiProject) getProjects(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	filter := storage.ProjectFilter{
		CreatorId: claims.Id,
	}
	var projects []storage.Project
	if err := a.db.GetList(&filter, &projects); err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	utils.ResponseOK(w, projects)
}
