package webserver

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
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
		log.Error(err)
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
