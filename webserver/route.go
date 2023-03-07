package webserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (s *WebServer) Route() {
	s.mux.Use(middleware.Recoverer, cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:6789", "http://localhost:8081"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	s.mux.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			var authRouter = apiAuth{WebServer: s}
			r.Post("/register", authRouter.register)
			r.Post("/login", authRouter.login)
			r.Post("/passkey/begin-login", authRouter.BeginLogin)
			r.Post("/passkey/finish-login", authRouter.FinishLogin)

			r.Group(func(r chi.Router) {
				r.Post("/verify-otp", authRouter.verifyOtp)
			})
		})
		r.Route("/user", func(r chi.Router) {
			r.Use(s.loggedInMiddleware)
			var userRouter = apiUser{WebServer: s}
			r.Get("/info", userRouter.info)
			r.Put("/info", userRouter.update)
			r.Get("/exist-checking", userRouter.checkingUserExist)
			r.Get("/generate-otp", userRouter.generateQr)
			r.Post("/disable-otp", userRouter.disableOtp)
			r.Post("/begin-registration", userRouter.BeginRegistration)
			r.Post("/finish-registration", userRouter.FinishRegistration)
		})

		r.Route("/admin", func(r chi.Router) {
			r.Use(s.loggedInMiddleware, s.adminMiddleware)
			r.Route("/user", func(r chi.Router) {
				var userRouter = apiUser{WebServer: s}
				r.Get("/info/{id}", userRouter.infoWithId)
				r.Put("/info", userRouter.adminUpdateUser)
				r.Get("/list", userRouter.getListUsers)
			})
		})
		r.Route("/payment", func(r chi.Router) {
			var paymentRouter = apiPayment{WebServer: s}
			r.With(s.loggedInMiddleware).Post("/", paymentRouter.createPayment)
			r.Get("/{id:[0-9]+}", paymentRouter.getPayment)
			r.Post("/{id:[0-9]+}", paymentRouter.updatePayment)
			r.Post("/request-rate", paymentRouter.requestRate)
			r.Post("/process", paymentRouter.processPayment)
			r.With(s.loggedInMiddleware).Get("/list", paymentRouter.listPayments)
		})
	})
}
