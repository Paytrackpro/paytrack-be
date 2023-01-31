package webserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (s *WebServer) Route() {
	s.mux.Use(middleware.Recoverer, cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
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
			r.Get("user/info/{id}", userRouter.infoWithId)
			r.Post("user/info", userRouter.update)
			r.Get("user/list", userRouter.getListUsers)
		})
	})
}
