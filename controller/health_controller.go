package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthController struct {
	db *pgxpool.Pool
}

func NewHealthController(db *pgxpool.Pool) *HealthController {
	return &HealthController{db: db}
}

type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Service  string `json:"service"`
	Time     string `json:"time"`
}

func (c *HealthController) CheckHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	status := "ok"
	dbStatus := "connected"
	statusCode := http.StatusOK

	if c.db != nil {
		if err := c.db.Ping(ctx); err != nil {
			status = "unhealthy"
			dbStatus = "disconnected"
			statusCode = http.StatusServiceUnavailable
		}
	} else {
		status = "unhealthy"
		dbStatus = "not_configured"
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(HealthResponse{
		Status:   status,
		Database: dbStatus,
		Service:  "talent-backend",
		Time:     time.Now().UTC().Format(time.RFC3339),
	})
}
