package webserver

import (
	"net/http"

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
	// The home route notifies that the API is up and running
	s.mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("MGMT-NG API is up and running"))
	})
	s.mux.Get("/socket.io/", s.handleSocket())
	s.mux.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			var authRouter = apiAuth{WebServer: s}
			r.Post("/register", authRouter.register)
			r.Post("/login", authRouter.login)

			r.Group(func(r chi.Router) {
				r.Use(s.loggedInMiddleware)
				r.Post("/verify-otp", authRouter.verifyOtp)
			})
		})
		r.Route("/user", func(r chi.Router) {
			r.Use(s.loggedInMiddleware)
			var userRouter = apiUser{WebServer: s}
			r.Get("/info", userRouter.info)
			r.Put("/info", userRouter.update)
			r.Put("/change-password", userRouter.changePassword)
			r.Put("/hide-paid", userRouter.hidePaid)
			r.Put("/show-approved", userRouter.showApproved)
			r.Get("/exist-checking", userRouter.checkingUserExist)
			r.Get("/get-user-list", userRouter.getUserSelectionList)
			r.Get("/exists", userRouter.usersExist)
			r.Get("/member-exist", userRouter.membersExist)
			r.Post("/generate-otp", userRouter.generateQr)
			r.Post("/disable-otp", userRouter.disableOtp)
			r.Route("/setting", func(r chi.Router) {
				r.Get("/payment", userRouter.getPaymentSetting)
				r.Put("/payment", userRouter.updatePaymentSetting)
			})
			r.Post("/start_timer", userRouter.startTimer)
			r.Post("/pause_timer", userRouter.pauseTimer)
			r.Post("/resume_timer", userRouter.resumeTimer)
			r.Post("/stop_timer", userRouter.stopTimer)
			r.Get("/get-running-timer", userRouter.getRunningTimer)
			r.Get("/get-time-log", userRouter.getTimeLogList)
			r.Put("/update-timer", userRouter.updateTimer)
			r.Delete("/timer-delete/{id:[0-9]+}", userRouter.deleteTimer)
		})

		r.Route("/file", func(r chi.Router) {
			r.Use(s.loggedInMiddleware)
			var fileRouter = apiFileUpload{WebServer: s}
			r.Post("/upload", fileRouter.uploadFiles)
			r.Post("/upload-one", fileRouter.uploadOneFile)
			r.Get("/base64", fileRouter.getProductImagesBase64)
			r.Get("/base64-one", fileRouter.getOneImageBase64)
			r.Get("/img-base64", fileRouter.getImageBase64)
		})

		r.Route("/admin", func(r chi.Router) {
			r.Use(s.loggedInMiddleware, s.adminMiddleware)
			var userRouter = apiUser{WebServer: s}
			r.Route("/user", func(r chi.Router) {
				r.Get("/info/{id}", userRouter.infoWithId)
				r.Put("/info", userRouter.adminUpdateUser)
				r.Get("/list", userRouter.getListUsers)
			})
			r.Get("/report-summary", userRouter.getAdminReportSummary)
		})
		r.Route("/payment", func(r chi.Router) {
			var paymentRouter = apiPayment{WebServer: s}
			r.With(s.loggedInMiddleware).Post("/", paymentRouter.createPayment)
			r.Get("/{id:[0-9]+}", paymentRouter.getPayment)
			r.Post("/{id:[0-9]+}", paymentRouter.updatePayment)
			r.Post("/request-rate", paymentRouter.requestRate)
			r.Post("/process", paymentRouter.processPayment)
			r.Post("/approve", paymentRouter.approveRequest)
			r.Post("/reject", paymentRouter.rejectPayment)
			r.Post("/bulk-paid-btc", paymentRouter.bulkPaidBTC)
			r.With(s.loggedInMiddleware).Get("/list", paymentRouter.listPayments)
			r.Get("/btc-bulk-rate", paymentRouter.getBtcBulkRate)
			r.Delete("/delete/{id:[0-9]+}", paymentRouter.deleteDraft)
			r.Get("/monthly-summary", paymentRouter.getMonthlySummary)
			r.Get("/initialization-count", paymentRouter.getInitializationCount)
			r.Get("/bulk-pay-count", paymentRouter.countBulkPayBTC)
			r.Get("/has-report", paymentRouter.hasReport)
			r.Get("/payment-report", paymentRouter.paymentReport)
			r.Get("/invoice-report", paymentRouter.invoiceReport)
			r.Get("/address-report", paymentRouter.addressReport)
			r.Get("/exchange-list", paymentRouter.getExchangeList)
			r.Get("/get-payment-users", paymentRouter.getPaymentUsers)
		})
		r.Route("/project", func(r chi.Router) {
			r.Use(s.loggedInMiddleware)
			var projectRouter = apiProject{WebServer: s}
			r.Post("/create", projectRouter.createProject)
			r.Get("/get-list", projectRouter.getProjects)
			r.Get("/get-my-project", projectRouter.getMyProjects)
			r.Put("/edit", projectRouter.editProject)
			r.Delete("/delete/{id:[0-9]+}", projectRouter.deleteProject)
		})
	})
}
