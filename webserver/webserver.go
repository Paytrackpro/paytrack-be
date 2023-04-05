package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"code.cryptopower.dev/mgmt-ng/be/email"
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"github.com/gorilla/schema"

	"code.cryptopower.dev/mgmt-ng/be/webserver/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v4"
)

type Config struct {
	Port              int    `yaml:"port"`
	HmacSecretKey     string `yaml:"hmacSecretKey"`
	AesSecretKey      string `yaml:"aesSecretKey"`
	AliveSessionHours int    `yaml:"aliveSessionHours"`
	ClientAddr        string `yaml:"clientAddr"`
}

type WebServer struct {
	mux       *chi.Mux
	conf      *Config
	db        storage.Storage
	validator *validator.Validate
	mail      *email.MailClient
	crypto    *utils.Cryptography
	service   *service.Service
}

const authClaimsCtxKey = "authClaimsCtxKey"

type Map map[string]interface{}

func NewWebServer(c Config, db storage.Storage, mailClient *email.MailClient) (*WebServer, error) {
	if c.Port == 0 {
		return nil, fmt.Errorf("please set up server port")
	}
	if c.AliveSessionHours <= 0 {
		return nil, fmt.Errorf("aliveSessionHours must be > 0")
	}
	if c.HmacSecretKey == "" {
		return nil, fmt.Errorf("please set up hmacSecretKey")
	}
	crypto, err := utils.NewCryptography(c.AesSecretKey)
	if err != nil {
		return nil, err
	}

	sv := service.NewService(db.GetDB())

	return &WebServer{
		mux:       chi.NewRouter(),
		conf:      &c,
		db:        db,
		validator: validator.New(),
		mail:      mailClient,
		crypto:    crypto,
		service:   sv,
	}, nil
}

func (s *WebServer) Run() error {
	s.Route()
	log.Info("mgmtng is running on port: %d", s.conf.Port)
	var server = http.Server{
		Addr:              fmt.Sprintf(":%d", s.conf.Port),
		Handler:           s.mux,
		TLSConfig:         nil,
		ReadTimeout:       0,
		ReadHeaderTimeout: 0,
		WriteTimeout:      0,
		IdleTimeout:       0,
		MaxHeaderBytes:    0,
		TLSNextProto:      nil,
		ConnState:         nil,
		ErrorLog:          nil,
		BaseContext:       nil,
		ConnContext:       nil,
	}
	return server.ListenAndServe()
}

func (s *WebServer) parseJSON(r *http.Request, data interface{}) error {
	if r.Body == nil {
		return utils.NewError(fmt.Errorf("body cannot be empty or nil"), utils.ErrorBodyRequited)
	}
	var decoder = json.NewDecoder(r.Body)
	var err = decoder.Decode(data)
	defer r.Body.Close()
	return err
}

func (s *WebServer) parseQuery(r *http.Request, data interface{}) error {
	// for POST request, we use json decoder. So here we just handle the case of GET request
	return schema.NewDecoder().Decode(data, r.URL.Query())
}

// parseQueryAndValidate parse the url query to a filter and validate the filter
// at the moment, only Sort filter is in need of validation
func (s *WebServer) parseQueryAndValidate(r *http.Request, data interface{}) error {
	// for POST request, we use json decoder. So here we just handle the case of GET request
	err := schema.NewDecoder().Decode(data, r.URL.Query())
	if err != nil {
		return err
	}
	var f, ok = data.(storage.Filter)
	if ok {
		return utils.ValidateSortField(f.Sortable(), f.RequestedSort())
	}
	return nil
}

func (s *WebServer) parseJSONAndValidate(r *http.Request, data interface{}) error {
	err := s.parseJSON(r, data)
	if err != nil {
		return err
	}
	return s.validator.Struct(data)
}

func (s *WebServer) loggedInMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var bearer = r.Header.Get("Authorization")
		// Should be a bearer token
		if len(bearer) > 6 && strings.ToUpper(bearer[0:7]) == "BEARER " {
			var tokenStr = bearer[7:]
			var claim authClaims
			token, err := jwt.ParseWithClaims(tokenStr, &claim, func(token *jwt.Token) (interface{}, error) {
				return []byte(s.conf.HmacSecretKey), nil
			})
			if err != nil {
				utils.Response(w, http.StatusBadRequest, utils.InvalidCredential, nil)
				return
			}
			ctx := context.WithValue(r.Context(), authClaimsCtxKey, token.Claims)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		utils.Response(w, http.StatusBadRequest, utils.InvalidCredential, nil)
	}
	return http.HandlerFunc(fn)
}

func (s *WebServer) adminMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		claims, _ := s.credentialsInfo(r)
		if claims.UserRole != utils.UserRoleAdmin {
			e := &utils.Error{
				Mess: "This api only for admin",
				Code: utils.ErrorForbidden,
			}
			utils.Response(w, http.StatusForbidden, e, nil)
			return
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func (s *WebServer) parseBearer(r *http.Request) (*authClaims, bool) {
	var bearer = r.Header.Get("Authorization")
	// Should be a bearer token
	if len(bearer) > 6 && strings.ToUpper(bearer[0:7]) == "BEARER " {
		var tokenStr = bearer[7:]
		var claim authClaims
		_, err := jwt.ParseWithClaims(tokenStr, &claim, func(token *jwt.Token) (interface{}, error) {
			return []byte(s.conf.HmacSecretKey), nil
		})
		if err != nil {
			return nil, false
		}
		return &claim, true
	}
	return nil, false
}

func (s *WebServer) credentialsInfo(r *http.Request) (*authClaims, bool) {
	val := r.Context().Value(authClaimsCtxKey)
	claims, ok := val.(*authClaims)
	return claims, ok
}
