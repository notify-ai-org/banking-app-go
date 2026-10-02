package api_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/notify-ai-org/banking-app-go/internal/api"
	"github.com/notify-ai-org/banking-app-go/internal/service"
	"github.com/notify-ai-org/banking-app-go/internal/store"
	"github.com/notify-ai-org/client-go/notify"
)

// fakeACP records what the SDK sends to acp-server.
type fakeACP struct {
	mu       sync.Mutex
	captures []map[string]any
	rules    []map[string]any
	models   []string
}

func (f *fakeACP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	defer f.mu.Unlock()
	switch r.URL.Path {
	case "/client/register":
		_ = json.NewEncoder(w).Encode(map[string]any{"token": "test-token"})
	case "/api/event":
		var list []map[string]any
		_ = json.Unmarshal(body, &list)
		f.captures = append(f.captures, list...)
	case "/api/vocabulary/rules/process":
		var rule map[string]any
		_ = json.Unmarshal(body, &rule)
		f.rules = append(f.rules, rule)
	case "/api/vocabulary":
		var models []map[string]any
		_ = json.Unmarshal(body, &models)
		for _, m := range models {
			f.models = append(f.models, m["className"].(string))
		}
	}
}

func (f *fakeACP) user(event string) map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.captures {
		ev := c["event"].(map[string]any)
		if ev["eventType"] == "USER" && ev["name"] == event {
			return c
		}
	}
	return nil
}

func TestBankingEventsReachACP(t *testing.T) {
	acp := &fakeACP{}
	acpSrv := httptest.NewServer(acp)
	defer acpSrv.Close()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := notify.New(notify.Config{ACPServerURL: acpSrv.URL, ClientToken: "client-test", FlushInterval: 10 * time.Millisecond, Logger: log})
	svc := service.New(client, store.NewAccountStore(), store.NewTransactionStore(), log)
	mux := http.NewServeMux()
	api.Routes(mux, svc)
	app := httptest.NewServer(mux)
	defer app.Close()

	if err := client.Start(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.WaitReady(ctx); err != nil {
		t.Fatal(err)
	}

	post := func(path, body string) map[string]any {
		resp, err := http.Post(app.URL+path, "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: status %d", path, resp.StatusCode)
		}
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return out
	}
	if out := post("/api/banking/suspicious-login", `{"userId":"ACC-1","ipAddress":"10.0.0.9","location":"Lagos"}`); out["status"] != "SUSPICIOUS_LOGIN_FLAGGED" {
		t.Errorf("suspicious-login reply = %v", out)
	}
	post("/api/banking/request-otp", `{"userId":"ACC-2","otpCode":"123456","channel":"SMS"}`)
	post("/api/banking/transfer", `{"transactionId":"TX-1","fromAccountId":"ACC-1","toAccountId":"ACC-2","amount":75000,"currency":"USD","type":"WIRE"}`)
	post("/api/banking/daily-summary", `{"accountId":"ACC-3","balance":250000.75,"currency":"USD","statementDate":"2026-10-01"}`)
	if err := client.Flush(ctx); err != nil {
		t.Fatal(err)
	}

	login := acp.user("SUSPICIOUS_LOGIN")
	subjects := login["subjectResult"].(map[string]any)["subjects"].([]any)
	if len(subjects) != 2 || subjects[0].(map[string]any)["channel"] != "SMS" || subjects[1].(map[string]any)["channel"] != "EMAIL" {
		t.Errorf("SUSPICIOUS_LOGIN subjects = %v", subjects)
	}
	if login["payload"].(map[string]any)["location"] != "Lagos" {
		t.Errorf("SUSPICIOUS_LOGIN payload = %v", login["payload"])
	}

	otp := acp.user("OTP_REQUESTED")["subjectResult"].(map[string]any)["subjects"].([]any)
	if len(otp) != 1 || otp[0].(map[string]any)["phoneNumber"] != "+1-555-0202" {
		t.Errorf("OTP_REQUESTED subjects = %v", otp)
	}

	transfer := acp.user("LARGE_TRANSFER")
	rules := map[string]any{}
	for _, r := range transfer["ruleResults"].([]any) {
		rules[r.(map[string]any)["ruleName"].(string)] = r.(map[string]any)["result"]
	}
	if rules["transfer-fraud-check"] != false || rules["velocity-check"] != true {
		t.Errorf("LARGE_TRANSFER rules = %v (75000 must fail the $50,000 fraud check)", rules)
	}

	summary := acp.user("DAILY_BALANCE_SUMMARY")
	if summary["payload"].(map[string]any)["statementDate"] != "2026-10-01" {
		t.Errorf("DAILY_BALANCE_SUMMARY payload = %v", summary["payload"])
	}

	acp.mu.Lock()
	defer acp.mu.Unlock()
	if len(acp.rules) != 2 {
		t.Errorf("rules registered = %v", acp.rules)
	}
	want := map[string]bool{"Account": true, "LoginPayload": true, "OtpPayload": true, "TransactionPayload": true, "BalanceSummaryPayload": true}
	for _, m := range acp.models {
		delete(want, m)
	}
	if len(want) != 0 {
		t.Errorf("models not registered: %v (got %v)", want, acp.models)
	}
}
