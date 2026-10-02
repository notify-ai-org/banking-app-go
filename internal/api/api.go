// Package api exposes HTTP endpoints that trigger banking events (port of the
// Java BankingController).
package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/notify-ai-org/banking-app-go/internal/model"
	"github.com/notify-ai-org/banking-app-go/internal/service"
)

// Routes registers the /api/banking endpoints.
func Routes(mux *http.ServeMux, s *service.TransactionService) {
	mux.HandleFunc("POST /api/banking/suspicious-login", handle(s.FlagSuspiciousLogin, func(p model.LoginPayload) any {
		return map[string]any{"status": "SUSPICIOUS_LOGIN_FLAGGED", "userId": p.UserID, "location": p.Location}
	}))
	mux.HandleFunc("POST /api/banking/request-otp", handle(s.RequestOtp, func(p model.OtpPayload) any {
		return map[string]any{"status": "OTP_SENT", "userId": p.UserID, "channel": p.Channel}
	}))
	mux.HandleFunc("POST /api/banking/transfer", handle(s.ProcessLargeTransfer, func(p model.TransactionPayload) any {
		return map[string]any{"status": "TRANSFER_INITIATED", "transactionId": p.TransactionID, "amount": p.Amount}
	}))
	mux.HandleFunc("POST /api/banking/daily-summary", handle(s.GenerateDailySummary, func(p model.BalanceSummaryPayload) any {
		return map[string]any{"status": "DAILY_SUMMARY_GENERATED", "accountId": p.AccountID, "balance": p.Balance}
	}))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func handle[P any](event func(context.Context, P) (P, error), reply func(P) any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p P
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if _, err := event(r.Context(), p); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, reply(p))
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
