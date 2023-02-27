package webserver

import (
	"errors"
	"fmt"
	"net/http"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"github.com/go-chi/chi/v5"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type apiUser struct {
	*WebServer
}

func (a *apiUser) info(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	user, err := a.db.QueryUser(storage.UserFieldId, claims.Id)
	if err != nil {
		utils.Response(w, http.StatusNotFound, err, nil)
	} else {
		utils.ResponseOK(w, user)
	}
}

func (a *apiUser) infoWithId(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := a.db.QueryUser(storage.UserFieldId, id)
	if err != nil {
		utils.Response(w, http.StatusNotFound, err, nil)
	} else {
		utils.ResponseOK(w, user)
	}
}

// This function use for update user for admin and user
func (a *apiUser) updateUser(w http.ResponseWriter, req portal.UpdateUserRequest) {

	user, err := a.db.QueryUser(storage.UserFieldId, req.UserId)
	if err != nil {
		utils.Response(w, http.StatusNotFound, err, nil)
		return
	}
	utils.SetValue(&user.DisplayName, req.DisplayName)
	utils.SetValue(&user.Email, req.Email)
	utils.SetValue(&user.Otp, req.Otp)
	user.PaymentSettings = req.PaymentSettings
	if !utils.IsEmpty(req.Password) {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			utils.Response(w, http.StatusInternalServerError, err, nil)
			return
		}
		user.PasswordHash = string(hash)
	}
	err = a.db.UpdateUser(user)

	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	utils.ResponseOK(w, Map{
		"userId": user.Id,
	})
}

func (a *apiUser) adminUpdateUser(w http.ResponseWriter, r *http.Request) {
	var f portal.UpdateUserRequest
	err := a.parseJSONAndValidate(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}
	a.updateUser(w, f)
}

func (a *apiUser) update(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)

	var f portal.UpdateUserRequest
	err := a.parseJSONAndValidate(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}
	f.UserId = int(claims.Id)
	a.updateUser(w, f)
}

func (a *apiUser) getListUsers(w http.ResponseWriter, r *http.Request) {
	var f storage.UserFilter
	if err := a.parseQueryAndValidate(r, &f); err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	var users []storage.User
	if err := a.db.GetList(&f, &users); err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
		return
	}
	count, _ := a.db.Count(&f, &storage.User{})
	utils.ResponseOK(w, Map{
		"users": users,
		"count": count,
	})
}

func (a *apiUser) checkingUserExist(w http.ResponseWriter, r *http.Request) {
	userName := r.FormValue("userName")
	claims, _ := a.credentialsInfo(r)
	if claims.UserName == userName {
		utils.ResponseOK(w, Map{
			"found":   false,
			"message": "userName must not be yours",
		})
		return
	}
	user, err := a.db.QueryUser(storage.UserFieldUName, userName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.ResponseOK(w, Map{
				"found":   false,
				"message": "userName not found",
			})
		} else {
			utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
		}
		return
	}
	utils.ResponseOK(w, Map{
		"found":           true,
		"id":              user.Id,
		"userName":        user.UserName,
		"paymentSettings": user.PaymentSettings,
	})
}

func (a *apiUser) generateQr(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	user, err := a.db.QueryUser(storage.UserFieldId, claims.Id)

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "MGMT",
		AccountName: user.UserName,
	})
	qrImage, err := key.Image(200, 200)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	imgBase64Str, err := utils.ImageToBase64(qrImage)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	utils.SetValue(&user.Secret, key.Secret())
	err = a.db.UpdateUser(user)

	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	utils.ResponseOK(w, Map{
		"mfa_qr_image": imgBase64Str,
	})
}

func (a *apiUser) disableOtp(w http.ResponseWriter, r *http.Request) {
	var f portal.OtpForm
	err := a.parseJSON(r, &f)

	claims, _ := a.credentialsInfo(r)

	user, err := a.db.QueryUser(storage.UserFieldId, claims.Id)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	verified := totp.Validate(f.Otp, user.Secret)

	if verified == false {
		err := utils.NewError(fmt.Errorf("OTP is not valid"), utils.ErrorObjectExist)
		utils.Response(w, http.StatusBadRequest, err, nil)

		return
	}

	utils.SetValue(&user.Otp, false)
	err = a.db.UpdateUser(user)

	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	utils.ResponseOK(w, Map{})
}
