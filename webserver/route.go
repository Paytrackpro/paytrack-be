package webserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (s *WebServer) Route() {
	s.mux.Use(middleware.Recoverer)
	s.mux.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			var authRouter = apiAuth{WebServer: s}
			r.Post("/register", authRouter.register)
			r.Post("/login", authRouter.login)
		})
		r.Route("/user", func(r chi.Router) {
			r.Use(s.loggedInMiddleware)
			var userRouter = apiUser{WebServer: s}
			r.Get("/info", userRouter.info)
			r.Post("/info", userRouter.update)
		})
	})
}
