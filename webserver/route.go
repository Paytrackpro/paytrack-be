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
			r.Get("/exist-checking", userRouter.checkingUserExist)
			r.Get("/exists", userRouter.usersExist)
			r.Post("/generate-otp", userRouter.generateQr)
			r.Post("/disable-otp", userRouter.disableOtp)
			r.Route("/setting", func(r chi.Router) {
				r.Get("/payment", userRouter.getPaymentSetting)
				r.Put("/payment", userRouter.updatePaymentSetting)
			})
		})

		r.Route("/shop", func(r chi.Router) {
			r.Use(s.loggedInMiddleware)
			r.Route("/product", func(r chi.Router) {
				var productRouter = apiProduct{WebServer: s}
				r.Post("/create", productRouter.createProduct)
				r.Get("/info/{id}", productRouter.info)
				r.Put("/update", productRouter.updateProduct)
				r.Get("/list", productRouter.getListProducts)
				r.Get("/store-list", productRouter.getListStore)
				r.Delete("/delete/{id:[0-9]+}", productRouter.deleteProduct)
			})
		})

		r.Route("/cart", func(r chi.Router) {
			r.Use(s.loggedInMiddleware)
			var apiRouter = apiCart{WebServer: s}
			r.Post("/add-to-cart", apiRouter.addToCart)
			r.Get("/list", apiRouter.getCartList)
			r.Get("/count", apiRouter.countCart)
			r.Put("/update", apiRouter.updateCart)
			r.Delete("/delete", apiRouter.deleteCart)
		})

		r.Route("/order", func(r chi.Router) {
			r.Use(s.loggedInMiddleware)
			var apiRouter = apiOrder{WebServer: s}
			r.Post("/createOrders", apiRouter.createOrders)
			r.Get("/order-management", apiRouter.getOrderManagement)
			r.Get("/detail/{id}", apiRouter.getOrderDetail)
			r.Get("/my-orders", apiRouter.getMyOrders)
		})

		r.Route("/file", func(r chi.Router) {
			r.Use(s.loggedInMiddleware)
			var fileRouter = apiFileUpload{WebServer: s}
			r.Post("/upload", fileRouter.uploadFile)
			r.Get("/base64", fileRouter.getProductImagesBase64)
			r.Get("/img-base64", fileRouter.getImageBase64)
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
			r.Post("/approve", paymentRouter.approveRequest)
			r.Post("/reject", paymentRouter.rejectPayment)
			r.Post("/bulk-paid-btc", paymentRouter.bulkPaidBTC)
			r.With(s.loggedInMiddleware).Get("/list", paymentRouter.listPayments)
			r.Get("/rate", paymentRouter.getRate)
			r.Delete("/delete/{id:[0-9]+}", paymentRouter.deleteDraft)
			r.Get("/monthly-summary", paymentRouter.getMonthlySummary)
			r.Get("/initialization-count", paymentRouter.getInitializationCount)
			r.Get("/bulk-pay-count", paymentRouter.countBulkPayBTC)
			r.Post("/delete-payment-product", paymentRouter.deletePaymentProduct)
		})
	})
}
