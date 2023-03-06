package webserver

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"github.com/go-webauthn/webauthn/webauthn"
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
	Id       uint64
	UserRole utils.UserRole
	Expire   int64
	UserName string
	Otp      bool
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
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	if err := a.db.CheckDuplicate(user); err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}
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

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(f.Password))
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.LoginFail, nil)
		return
	}

	if user.Otp {
		utils.ResponseOK(w, Map{
			"userId": user.Id,
			"otp":    true,
		})
		return
	}

	var authClaim = authClaims{
		Id:       user.Id,
		UserRole: user.Role,
		Expire:   time.Now().Add(time.Hour * time.Duration(a.conf.AliveSessionHours)).Unix(),
		UserName: user.UserName,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, authClaim)
	tokenString, err := token.SignedString([]byte(a.conf.HmacSecretKey))
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
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

	user, err := a.db.QueryUser(storage.UserFieldId, f.UserId)
	if err != nil {
		err := utils.NewError(fmt.Errorf("OTP is not valid"), utils.ErrorObjectExist)
		utils.Response(w, http.StatusBadRequest, err, nil)

		return
	}

	verified := totp.Validate(f.Otp, user.Secret)

	if verified == false {
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
		Id:       user.Id,
		UserRole: user.Role,
		Expire:   time.Now().Add(time.Hour * time.Duration(a.conf.AliveSessionHours)).Unix(),
		UserName: user.UserName,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, authClaim)
	tokenString, err := token.SignedString([]byte(a.conf.HmacSecretKey))
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	utils.ResponseOK(w, Map{
		"token":    tokenString,
		"userInfo": user,
	})
}

func (a *apiAuth) BeginLogin(w http.ResponseWriter, r *http.Request) {
	var f portal.PasskeyLoginForm
	err := a.parseJSON(r, &f)

	user, err := a.db.QueryUser(storage.UserFieldUName, f.UserName)
	if err != nil {
		utils.Response(w, http.StatusNotFound, err, nil)
		return
	}
	options, sessionData, err := a.webAuthn.BeginLogin(user)

	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	// store session data as marshaled JSON
	session, err := a.sessionStore.New(r, "authentication")
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	session.Values["sessionData"] = sessionData
	err = session.Save(r, w)

	// store session data as marshaled JSON
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	utils.ResponseOK(w, Map{
		"options": options,
	})
}

func (a *apiAuth) FinishLogin(w http.ResponseWriter, r *http.Request) {
	var f portal.PasskeyLoginForm
	err := a.parseJSON(r, &f)

	user, err := a.db.QueryUser(storage.UserFieldUName, f.UserName)
	if err != nil {
		utils.Response(w, http.StatusNotFound, err, nil)
		return
	}

	// load the session data
	session, err := a.sessionStore.Get(r, "authentication")
	var sessionData = &webauthn.SessionData{}
	var ok bool

	if sessionData, ok = session.Values["sessionData"].(*webauthn.SessionData); !ok {
		utils.Response(w, http.StatusNotFound, err, nil)
		return
	}

	// // in an actual implementation, we should perform additional checks on
	// // the returned 'credential', i.e. check 'credential.Authenticator.CloneWarning'
	// // and then increment the credentials counter
	_, err = a.webAuthn.FinishLogin(user, *sessionData, r)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	// // handle successful login
	// jsonResponse(w, "Login Success", http.StatusOK)

	utils.ResponseOK(w, Map{})
}
