package webserver

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"code.cryptopower.dev/mgmt-ng/be/webserver/service"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type apiAuth struct {
	*WebServer
}

type authClaims struct {
	Id                    uint64
	UserRole              utils.UserRole
	Expire                int64
	UserName              string
	DisplayName           string
	Otp                   bool
	ShowDraftForRecipient bool
	ShowDateOnInvoiceLine bool
	HidePaid              bool
}

func (c authClaims) Valid() error {
	timestamp := time.Now().Unix()
	if timestamp >= c.Expire {
		return fmt.Errorf("the credential is expired")
	}
	return nil
}

func (a *apiAuth) register(w http.ResponseWriter, r *http.Request) {
	var f portal.RegisterForm
	err := a.parseJSONAndValidate(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}
	user, err := f.User()
	if err != nil {
		log.Error(err)
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	if err := a.db.CheckDuplicate(user); err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}
	//default: set display date on invoice is true
	user.ShowDateOnInvoiceLine = true
	user.ShowDraftForRecipient = true
	if err := a.db.CreateUser(user); err != nil {
		// 23505 is  duplicate key value error for postgresql
		if e, ok := err.(*pgconn.PgError); ok && e.Code == utils.PgsqlDuplicateErrorCode {
			if e.ConstraintName == "users_user_name_idx" {
				utils.Response(w, http.StatusBadRequest,
					utils.NewError(fmt.Errorf("the user name '%s' is already taken", f.UserName),
						utils.ErrorObjectExist), nil)
				return
			}
		}
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	utils.ResponseOK(w, Map{
		"userId": user.Id,
	})
}

func (a *apiAuth) CancelPasskeyRegister(w http.ResponseWriter, r *http.Request) {
	sessionKey := r.FormValue("sessionKey")
	var response utils.ResponseData
	if err := service.HttpPost(fmt.Sprintf("%s%s", a.conf.AuthHost, "/passkey/cancelRegister"), url.Values{"sessionKey": {sessionKey}}, &response); err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	if response.IsError {
		utils.Response(w, http.StatusInternalServerError, fmt.Errorf(response.Msg), nil)
		return
	}
	utils.ResponseOK(w, response.Data)
}

func (a *apiAuth) StartPasskeyRegister(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	var response utils.ResponseData
	if err := service.HttpPost(fmt.Sprintf("%s%s", a.conf.AuthHost, "/passkey/registerStart"), url.Values{"username": {username}}, &response); err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	if response.IsError {
		utils.Response(w, http.StatusInternalServerError, fmt.Errorf(response.Msg), nil)
		return
	}
	utils.ResponseOK(w, response.Data)
}

func (a *apiAuth) FinishPasskeyRegister(w http.ResponseWriter, r *http.Request) {
	reqUrl := fmt.Sprintf("%s%s", a.conf.AuthHost, "/passkey/registerFinish")
	Headers := map[string]string{
		"Content-Type": "application/json",
		"Session-Key":  r.FormValue("sessionKey"),
	}
	var response utils.ResponseData
	if err := service.HttpFullPost(reqUrl, r.Body, Headers, &response); err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	if response.IsError {
		utils.Response(w, http.StatusInternalServerError, fmt.Errorf(response.Msg), nil)
		return
	}
	//get display name
	displayName := r.FormValue("dispName")
	email := r.FormValue("email")
	data, isOk := response.Data.(map[string]any)
	tokenString := ""
	var authClaim storage.AuthClaims
	if isOk {
		tokenString, _ = data["token"].(string)
		err := utils.CatchObject(data["user"], &authClaim)
		if err != nil {
			utils.Response(w, http.StatusInternalServerError, err, nil)
			return
		}
	} else {
		utils.Response(w, http.StatusInternalServerError, fmt.Errorf("No data returned"), nil)
		return
	}

	//insert to user settings in local db
	var user = storage.User{
		Id:                    uint64(authClaim.Id),
		UserName:              authClaim.Username,
		DisplayName:           displayName,
		Email:                 email,
		ShowDraftForRecipient: true,
		ShowDateOnInvoiceLine: true,
		LastSeen:              time.Now(),
	}
	if err := a.db.CreateUser(&user); err != nil {
		// 23505 is  duplicate key value error for postgresql
		if e, ok := err.(*pgconn.PgError); ok && e.Code == utils.PgsqlDuplicateErrorCode {
			if e.ConstraintName == "users_user_name_idx" {
				utils.Response(w, http.StatusBadRequest,
					utils.NewError(fmt.Errorf("the user name is already taken"),
						utils.ErrorObjectExist), nil)
				return
			}
		}
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	//handler login imediately
	utils.ResponseOK(w, Map{
		"token":    tokenString,
		"userInfo": user,
	})
}

func (a *apiAuth) CheckingUsernameExist(w http.ResponseWriter, r *http.Request) {
	userName := r.FormValue("userName")
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
		"found":    true,
		"id":       user.Id,
		"userName": user.UserName,
	})
}

func (a *apiAuth) AssertionResult(w http.ResponseWriter, r *http.Request) {
	var response utils.ResponseData
	reqUrl := fmt.Sprintf("%s%s", a.conf.AuthHost, "/assertion/result")
	Headers := map[string]string{
		"Content-Type": "application/json",
		"Session-Key":  r.FormValue("sessionKey"),
	}
	if err := service.HttpFullPost(reqUrl, r.Body, Headers, &response); err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	if response.IsError {
		utils.Response(w, http.StatusInternalServerError, fmt.Errorf(response.Msg), nil)
		return
	}
	data, isOk := response.Data.(map[string]any)
	tokenString := ""
	var authClaim storage.AuthClaims
	if isOk {
		tokenString, _ = data["token"].(string)
		err := utils.CatchObject(data["user"], &authClaim)
		if err != nil {
			utils.Response(w, http.StatusInternalServerError, err, nil)
			return
		}
	} else {
		utils.Response(w, http.StatusInternalServerError, fmt.Errorf("No data returned"), nil)
		return
	}
	//Get user settings from local DB
	user, err := a.db.QueryUser(storage.UserFieldUName, authClaim.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Response(w, http.StatusNotFound, utils.InvalidCredential, nil)
		} else {
			utils.Response(w, http.StatusInternalServerError, err, nil)
		}
		return
	}

	//If the user is locked, can't login
	if user.Locked {
		err := utils.NewError(fmt.Errorf("user has been locked"), utils.ErrorObjectExist)
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	utils.ResponseOK(w, Map{
		"token":    tokenString,
		"userInfo": user,
	})
}

func (a *apiAuth) AssertionOptions(w http.ResponseWriter, r *http.Request) {
	var response utils.ResponseData
	if err := service.HttpPost(fmt.Sprintf("%s%s", a.conf.AuthHost, "/assertion/options"), url.Values{}, &response); err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	utils.ResponseOK(w, response)
}

func (a *apiAuth) getAuthMethod(w http.ResponseWriter, r *http.Request) {
	authType := a.conf.AuthType
	if authType != int(storage.AuthLocalUsernamePassword) && authType != int(storage.AuthMicroservicePasskey) {
		authType = int(storage.AuthLocalUsernamePassword)
	}
	utils.ResponseOK(w, authType)
}

func (a *apiAuth) login(w http.ResponseWriter, r *http.Request) {
	var f portal.LoginForm
	err := a.parseJSON(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	user, err := a.db.QueryUser(storage.UserFieldUName, f.UserName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Response(w, http.StatusNotFound, utils.LoginFail, nil)
		} else {
			utils.Response(w, http.StatusInternalServerError, err, nil)
		}
		return
	}

	//If the user is locked, can't login
	if user.Locked {
		err := utils.NewError(fmt.Errorf("user has been locked"), utils.ErrorObjectExist)
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(f.Password))
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.LoginFail, nil)
		return
	}

	if f.IsOtp {
		verified := totp.Validate(f.Otp, user.Secret)

		if !verified {
			err := utils.NewError(fmt.Errorf("OTP is not valid"), utils.ErrorObjectExist)
			utils.Response(w, http.StatusBadRequest, err, nil)

			return
		}
	}

	if user.Otp && !f.IsOtp {
		utils.ResponseOK(w, Map{
			"userId": user.Id,
			"otp":    true,
		})
		return
	}

	var authClaim = authClaims{
		Id:                    user.Id,
		UserRole:              user.Role,
		Expire:                time.Now().Add(time.Hour * time.Duration(a.conf.AliveSessionHours)).Unix(),
		UserName:              user.UserName,
		DisplayName:           user.DisplayName,
		ShowDraftForRecipient: user.ShowDraftForRecipient,
		ShowDateOnInvoiceLine: user.ShowDateOnInvoiceLine,
		HidePaid:              user.HidePaid,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, authClaim)
	tokenString, err := token.SignedString([]byte(a.conf.HmacSecretKey))
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	//update last seen for User
	a.service.SetLastSeen(int(user.Id))
	utils.ResponseOK(w, Map{
		"token":    tokenString,
		"userInfo": user,
	})
}

func (a *apiAuth) verifyOtp(w http.ResponseWriter, r *http.Request) {
	var f portal.OtpForm
	err := a.parseJSON(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	claims, _ := a.credentialsInfo(r)
	user, err := a.db.QueryUser(storage.UserFieldUName, claims.UserName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.Response(w, http.StatusNotFound, utils.InvalidCredential, nil)
			return
		}
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(f.Password))
	if err != nil {
		log.Error(err)
		utils.Response(w, http.StatusBadRequest, utils.InvalidCredential, nil)
		return
	}

	verified := totp.Validate(f.Otp, user.Secret)

	if !verified {
		err := utils.NewError(fmt.Errorf("OTP is not valid"), utils.ErrorObjectExist)
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	if f.FirstTime {
		utils.SetValue(&user.Otp, f.FirstTime)
		err := a.db.UpdateUser(user)

		if err != nil {
			utils.Response(w, http.StatusInternalServerError, err, nil)
			return
		}
	}

	var authClaim = authClaims{
		Id:                    user.Id,
		UserRole:              user.Role,
		Expire:                time.Now().Add(time.Hour * time.Duration(a.conf.AliveSessionHours)).Unix(),
		UserName:              user.UserName,
		DisplayName:           user.DisplayName,
		ShowDraftForRecipient: user.ShowDraftForRecipient,
		ShowDateOnInvoiceLine: user.ShowDateOnInvoiceLine,
		HidePaid:              user.HidePaid,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, authClaim)
	tokenString, err := token.SignedString([]byte(a.conf.HmacSecretKey))
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	//update last seen for User
	a.service.SetLastSeen(int(user.Id))
	utils.ResponseOK(w, Map{
		"token":    tokenString,
		"userInfo": user,
	})
}
