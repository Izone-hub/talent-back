package router

import (
	"net/http"

	"github.com/Izone-hub/talent-backend/controller"
	"github.com/Izone-hub/talent-backend/middleware"
)

type Router struct {
	authController *controller.AuthController
	authMiddleware *middleware.AuthMiddleware
}

func NewRouter(
	authController *controller.AuthController,
	authMiddleware *middleware.AuthMiddleware,
) *Router {
	return &Router{
		authController: authController,
		authMiddleware: authMiddleware,
	}
}

func (r *Router) Initialize() http.Handler {
	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("GET /api/auth/github/login", r.authController.GitHubLogin)
	mux.HandleFunc("GET /api/auth/github/callback", r.authController.GitHubCallback)

	// Protected routes
	mux.HandleFunc("GET /api/auth/me", r.authMiddleware.Authenticate(r.authController.GetCurrentUser))

	// Example admin route
	mux.HandleFunc("GET /api/admin/dashboard",
		r.authMiddleware.Authenticate(
			r.authMiddleware.RequireAdmin(
				func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Admin Dashboard"))
				},
			),
		),
	)

	return mux
}
