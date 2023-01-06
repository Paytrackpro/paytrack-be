package webserver

import (
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v4"
	"log"
	"net/http"
	"strings"
)

type Config struct {
	Port          int    `yaml:"port"`
	HmacSecretKey string `yaml:"hmacSecretKey"`
}

type Response struct {
	httpCode int
	Success  bool        `json:"success"`
	Error    string      `json:"error,omitempty"`
	Data     interface{} `json:"data,omitempty"`
}

type WebServer struct {
	mux  *chi.Mux
	conf *Config
	db   storage.Storage
}

type Map map[string]interface{}

func NewWebServer(c Config, db storage.Storage) (*WebServer, error) {
	c.HmacSecretKey = hmacDefaultSecret
	return &WebServer{
		mux:  chi.NewRouter(),
		conf: &c,
		db:   db,
	}, nil
}

func (s *WebServer) Run() error {
	s.Route()
	log.Printf("mgmtngd is running on :%d", s.conf.Port)
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

func (s *WebServer) response(data Response, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(data.httpCode)
	body, _ := json.Marshal(data)
	w.Write(body)
}

func (s *WebServer) parseJSON(r *http.Request, data interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("body is nil")
	}
	var decoder = json.NewDecoder(r.Body)
	var err = decoder.Decode(data)
	defer r.Body.Close()
	return err
}

func (s *WebServer) errorResponse(w http.ResponseWriter, err error, code int) {
	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	s.response(Response{
		httpCode: code,
		Success:  false,
		Error:    errStr,
		Data:     nil,
	}, w)
}

func (s *WebServer) successResponse(w http.ResponseWriter, data interface{}) {
	s.response(Response{
		httpCode: http.StatusOK,
		Success:  true,
		Error:    "",
		Data:     data,
	}, w)
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
				s.errorResponse(w, fmt.Errorf("your credential is invalid"), http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), authClaimsCtxKey, token.Claims)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		s.errorResponse(w, fmt.Errorf("your credential is invalid"), http.StatusUnauthorized)
	}
	return http.HandlerFunc(fn)
}

func (s *WebServer) credentialsInfo(r *http.Request) (*authClaims, bool) {
	val := r.Context().Value(authClaimsCtxKey)
	claims, ok := val.(*authClaims)
	return claims, ok
}
