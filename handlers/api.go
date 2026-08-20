package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"saas/auth"
	"saas/middleware"
	"saas/store"
)

const maxBodyBytes = 1 << 20

type APIHandler struct {
	store *store.Store
}

func NewAPIHandler(db *store.Store) *APIHandler {
	return &APIHandler{store: db}
}

func (h *APIHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*store.User)
	writeJSON(w, http.StatusOK, user)
}

func (h *APIHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*store.User)

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email is required"})
		return
	}

	updated, err := h.store.UpdateUser(user.ID, req.Name, req.Email)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "email already in use"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not update profile"})
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *APIHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*store.User)
	if err := h.store.DeleteUser(user.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not delete account"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "account deleted"})
}

func (h *APIHandler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*store.User)

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req struct {
		Plan string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	validPlans := map[string]bool{"free": true, "pro": true, "enterprise": true}
	if !validPlans[req.Plan] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid plan, choose: free, pro, enterprise"})
		return
	}

	if err := h.store.UpdateUserPlan(user.ID, req.Plan); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not update plan"})
		return
	}

	user.Plan = req.Plan
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "plan updated",
		"user":    user,
	})
}

func (h *APIHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*store.User)

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if req.Name == "" {
		req.Name = "default"
	}

	key, err := auth.GenerateAPIKey()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate key"})
		return
	}

	apiKey, err := h.store.CreateAPIKey(user.ID, key, req.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save key"})
		return
	}

	writeJSON(w, http.StatusCreated, apiKey)
}

func (h *APIHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*store.User)
	keys, err := h.store.GetAPIKeysByUser(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list keys"})
		return
	}
	if keys == nil {
		keys = []store.APIKey{}
	}
	writeJSON(w, http.StatusOK, keys)
}

func (h *APIHandler) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*store.User)

	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid key id"})
		return
	}

	if err := h.store.DeleteAPIKey(id, user.ID); err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "key not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete key"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "key deleted"})
}

var monitorLimits = map[string]int{
	"free":       5,
	"pro":        50,
	"enterprise": -1,
}

func (h *APIHandler) CreateMonitor(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*store.User)

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Type     string `json:"type"`
		Interval int    `json:"interval"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.URL = strings.TrimSpace(req.URL)
	if req.Name == "" || req.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and url are required"})
		return
	}
	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	if req.Type == "" {
		req.Type = "http"
	}
	validTypes := map[string]bool{"http": true, "tcp": true, "dns": true, "chain": true}
	if !validTypes[req.Type] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid type, choose: http, tcp, dns, chain"})
		return
	}
	validIntervals := map[int]bool{10: true, 30: true, 60: true, 300: true}
	if !validIntervals[req.Interval] {
		req.Interval = 60
	}

	limit := monitorLimits[user.Plan]
	if limit >= 0 {
		count, err := h.store.CountMonitorsByUser(user.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		if count >= limit {
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error": "monitor limit reached for " + user.Plan + " plan (max " + strconv.Itoa(limit) + ")",
			})
			return
		}
	}

	monitor, err := h.store.CreateMonitor(user.ID, req.Name, req.URL, req.Type, req.Interval)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not create monitor"})
		return
	}
	writeJSON(w, http.StatusCreated, monitor)
}

func (h *APIHandler) ListMonitors(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*store.User)
	monitors, err := h.store.GetMonitorsByUser(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list monitors"})
		return
	}
	if monitors == nil {
		monitors = []store.Monitor{}
	}
	writeJSON(w, http.StatusOK, monitors)
}

func (h *APIHandler) DeleteMonitor(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*store.User)

	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid monitor id"})
		return
	}

	if err := h.store.DeleteMonitor(id, user.ID); err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "monitor not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete monitor"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "monitor deleted"})
}

func (h *APIHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*store.User)

	count, _ := h.store.CountMonitorsByUser(user.ID)
	limit := monitorLimits[user.Plan]

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user":           user,
		"plan":           user.Plan,
		"monitors_count": count,
		"monitor_limit":  limit,
	})
}
