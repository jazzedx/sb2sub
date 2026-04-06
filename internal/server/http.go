package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"sb2sub/internal/config"
	"sb2sub/internal/model"
	"sb2sub/internal/render"
	"sb2sub/internal/service"
)

type Handler struct {
	config  config.Config
	service *service.Service
	mux     *http.ServeMux
}

func NewHandler(cfg config.Config, svc *service.Service) http.Handler {
	h := &Handler{
		config:  cfg,
		service: svc,
		mux:     http.NewServeMux(),
	}

	h.mux.HandleFunc("/healthz", h.handleHealth)
	h.mux.HandleFunc("/api/users", h.handleUsers)
	h.mux.HandleFunc("/api/subscriptions", h.handleSubscriptions)
	h.mux.HandleFunc("/sub/", h.handleSubscription)
	h.mux.HandleFunc("/", h.handleSubscription)

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users, err := h.service.ListUsers()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, users)
	case http.MethodPost:
		var payload struct {
			Username          string `json:"username"`
			Note              string `json:"note"`
			Enabled           bool   `json:"enabled"`
			QuotaBytes        int64  `json:"quota_bytes"`
			ExpiresAt         string `json:"expires_at"`
			VLESSUUID         string `json:"vless_uuid"`
			Hysteria2Password string `json:"hysteria2_password"`
			VLESSEnabled      bool   `json:"vless_enabled"`
			Hysteria2Enabled  bool   `json:"hysteria2_enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}

		expiresAt, err := time.Parse(time.RFC3339, payload.ExpiresAt)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid expires_at"})
			return
		}

		user, err := h.service.CreateUser(model.User{
			Username:          payload.Username,
			Note:              payload.Note,
			Enabled:           payload.Enabled,
			QuotaBytes:        payload.QuotaBytes,
			ExpiresAt:         expiresAt,
			VLESSUUID:         payload.VLESSUUID,
			Hysteria2Password: payload.Hysteria2Password,
			VLESSEnabled:      payload.VLESSEnabled,
			Hysteria2Enabled:  payload.Hysteria2Enabled,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, user)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	requestPath := strings.TrimPrefix(r.URL.Path, "/")
	if requestPath == "" {
		http.NotFound(w, r)
		return
	}

	subscription, err := h.findSubscription(requestPath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !subscription.Enabled {
		http.NotFound(w, r)
		return
	}

	user, err := h.service.GetUserByID(subscription.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !user.Enabled || user.ExpiresAt.Before(time.Now().UTC()) {
		http.NotFound(w, r)
		return
	}

	var body []byte
	switch subscription.Type {
	case model.SubscriptionTypeShadowrocket:
		body, err = render.RenderShadowrocket(h.config, user)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	case model.SubscriptionTypeClash, model.SubscriptionTypeBoth:
		body, err = render.RenderClash(h.config, user)
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported subscription type"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	_ = h.service.TouchSubscriptionAccess(subscription.ID)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (h *Handler) findSubscription(requestPath string) (model.Subscription, error) {
	if strings.HasPrefix(requestPath, "sub/") {
		token := strings.TrimPrefix(requestPath, "sub/")
		subscription, err := h.service.GetSubscriptionByToken(token)
		if err == nil {
			return subscription, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return model.Subscription{}, err
		}
	}

	return h.service.GetSubscriptionByCustomPath(requestPath)
}

func (h *Handler) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var payload struct {
			UserID     int64  `json:"user_id"`
			Name       string `json:"name"`
			Type       string `json:"type"`
			Token      string `json:"token"`
			CustomPath string `json:"custom_path"`
			Enabled    bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}

		subscription, err := h.service.CreateSubscription(model.Subscription{
			UserID:     payload.UserID,
			Name:       payload.Name,
			Type:       model.SubscriptionType(payload.Type),
			Token:      payload.Token,
			CustomPath: payload.CustomPath,
			Enabled:    payload.Enabled,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, subscription)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
