package webserver

import (
	"net/http"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
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
	memberArr := make(storage.Members, 0)
	memberArr = append(memberArr, storage.Member{
		MemberId:    claims.Id,
		UserName:    claims.UserName,
		DisplayName: claims.DisplayName,
		Role:        int(claims.UserRole),
	})
	for _, member := range body.Members {
		if member.MemberId == claims.Id {
			continue
		}
		memberArr = append(memberArr, member)
	}
	body.Members = memberArr
	creatorName := claims.UserName
	if !utils.IsEmpty(claims.DisplayName) {
		creatorName = claims.DisplayName
	}
	project, err := a.service.CreateNewProject(claims.Id, creatorName, body)
	if err != nil {
		utils.Response(w, http.StatusForbidden, utils.NewError(err, utils.ErrorForbidden), nil)
	}
	utils.ResponseOK(w, project)
}

func (a *apiProject) getProjects(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	projects, err := a.service.GetMyProjects(claims.Id)
	if err != nil && err == gorm.ErrRecordNotFound {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	utils.ResponseOK(w, projects)
}

func (a *apiProject) getMyProjects(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)

	projects, err := a.service.GetMyProjects(claims.Id)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	utils.ResponseOK(w, projects)
}

func (a *apiProject) editProject(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)

	var body portal.ProjectRequest
	err := a.parseJSONAndValidate(r, &body)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}
	//if projectId not exist
	if body.ProjectId < 1 {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	memberArr := make(storage.Members, 0)
	memberArr = append(memberArr, storage.Member{
		MemberId:    claims.Id,
		UserName:    claims.UserName,
		DisplayName: claims.DisplayName,
		Role:        int(claims.UserRole),
	})
	for _, member := range body.Members {
		if member.MemberId == claims.Id {
			continue
		}
		memberArr = append(memberArr, member)
	}
	body.Members = memberArr
	project, err := a.service.UpdateProject(claims.Id, body)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	utils.ResponseOK(w, project)
}

func (a *apiProject) deleteProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	a.db.GetDB().Where("project_id = ?", id).Delete(&storage.Project{})
	utils.ResponseOK(w, nil)
}
