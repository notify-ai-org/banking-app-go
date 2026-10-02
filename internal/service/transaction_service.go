// Package service is the core banking service. It is the Go port of the Java
// TransactionService and exercises every Notify SDK registration type.
//
// Events:
//   - SUSPICIOUS_LOGIN (critical SMS + email)
//   - OTP_REQUESTED (immediate SMS)
//   - LARGE_TRANSFER (email with fraud + velocity rules)
//   - DAILY_BALANCE_SUMMARY (scheduled daily digest)
package service

import (
	"context"
	"log/slog"

	"github.com/notify-ai-org/banking-app-go/internal/model"
	"github.com/notify-ai-org/banking-app-go/internal/store"
	"github.com/notify-ai-org/client-go/notify"
)

// TransactionService holds the business logic. Its exported event fields are
// the notify-wrapped versions of the methods below; callers use those.
type TransactionService struct {
	accounts     *store.AccountStore
	transactions *store.TransactionStore
	log          *slog.Logger

	FlagSuspiciousLogin  func(context.Context, model.LoginPayload) (model.LoginPayload, error)
	RequestOtp           func(context.Context, model.OtpPayload) (model.OtpPayload, error)
	ProcessLargeTransfer func(context.Context, model.TransactionPayload) (model.TransactionPayload, error)
	GenerateDailySummary func(context.Context, model.BalanceSummaryPayload) (model.BalanceSummaryPayload, error)
}

// New builds the service and registers its events, subject suppliers,
// vocabulary supplier, rules and callbacks with the client.
func New(client *notify.Client, accounts *store.AccountStore, transactions *store.TransactionStore, log *slog.Logger) *TransactionService {
	s := &TransactionService{accounts: accounts, transactions: transactions, log: log}
	notify.RegisterModel[model.Account](client, model.Account{}.NotifyModelDescription())

	// ═══ EVENTS ═══
	s.FlagSuspiciousLogin = notify.Event(client, notify.EventSpec{Key: "SUSPICIOUS_LOGIN",
		Description: "Suspicious login attempt detected on an account", EventType: "static",
		ScheduleIntent: "immediate", PreferredTimeWindow: "00:00-23:59", Priority: 5}, s.flagSuspiciousLogin)
	s.RequestOtp = notify.Event(client, notify.EventSpec{Key: "OTP_REQUESTED",
		Description: "User requested a one-time password", EventType: "static",
		ScheduleIntent: "immediate", PreferredTimeWindow: "00:00-23:59", Priority: 5}, s.requestOtp)
	s.ProcessLargeTransfer = notify.Event(client, notify.EventSpec{Key: "LARGE_TRANSFER",
		Description: "Large fund transfer initiated", EventType: "static",
		ScheduleIntent: "immediate", PreferredTimeWindow: "09:00-17:00", Priority: 4}, s.processLargeTransfer)
	s.GenerateDailySummary = notify.Event(client, notify.EventSpec{Key: "DAILY_BALANCE_SUMMARY",
		Description: "Daily account balance summary for digest notifications", EventType: "deferred",
		ScheduleIntent: "scheduled", PreferredTimeWindow: "06:00-08:00", Priority: 2}, s.generateDailySummary)

	// ═══ SUBJECT SUPPLIERS ═══
	notify.SubjectSupplier(client, "SUSPICIOUS_LOGIN", "Resolves account holder to SMS + email for security alerts", s.suspiciousLoginSubjects)
	notify.SubjectSupplier(client, "OTP_REQUESTED", "Resolves user to SMS for OTP delivery", s.otpSubjects)
	notify.SubjectSupplier(client, "LARGE_TRANSFER", "Resolves sender account to email for transfer confirmation", s.transferSubjects)
	notify.SubjectSupplier(client, "DAILY_BALANCE_SUMMARY", "Resolves account to email for daily digest", s.dailySummarySubjects)

	// ═══ VOCABULARY SUPPLIER ═══
	notify.VocabularySupplier(client, "LARGE_TRANSFER", "Enriches transfer payload with sender and receiver names", s.transferVocabulary)

	// ═══ RULES ═══
	notify.Rule(client, notify.RuleSpec{Name: "transfer-fraud-check", Event: "LARGE_TRANSFER",
		Description: "Blocks transfers over $50,000 as potential fraud"}, s.fraudCheckRule)
	notify.Rule(client, notify.RuleSpec{Name: "velocity-check", Event: "LARGE_TRANSFER",
		Description: "Blocks if sender has more than 5 recent transactions"}, s.velocityCheckRule)

	// ═══ CALLBACKS ═══
	notify.Callback(client, "SUSPICIOUS_LOGIN", notify.Before, s.beforeSuspiciousLogin)
	notify.Callback(client, "LARGE_TRANSFER", notify.After, s.afterLargeTransfer)
	return s
}

// ── event bodies ─────────────────────────────────────────────────────────────

func (s *TransactionService) flagSuspiciousLogin(_ context.Context, p model.LoginPayload) (model.LoginPayload, error) {
	s.log.Warn("Suspicious login detected", "user", p.UserID, "ip", p.IPAddress, "location", p.Location)
	return p, nil
}

func (s *TransactionService) requestOtp(_ context.Context, p model.OtpPayload) (model.OtpPayload, error) {
	s.log.Info("OTP requested", "user", p.UserID, "channel", p.Channel)
	return p, nil
}

func (s *TransactionService) processLargeTransfer(_ context.Context, p model.TransactionPayload) (model.TransactionPayload, error) {
	s.log.Info("Large transfer", "from", p.FromAccountID, "to", p.ToAccountID, "amount", p.Amount, "currency", p.Currency)
	s.transactions.Save(p)
	return p, nil
}

func (s *TransactionService) generateDailySummary(_ context.Context, p model.BalanceSummaryPayload) (model.BalanceSummaryPayload, error) {
	s.log.Info("Daily summary", "account", p.AccountID, "balance", p.Balance, "currency", p.Currency)
	return p, nil
}

// ── subject suppliers ────────────────────────────────────────────────────────

func holder(a model.Account) map[string]string { return map[string]string{"holderName": a.HolderName} }

func (s *TransactionService) suspiciousLoginSubjects(_ context.Context, p model.LoginPayload) ([]notify.Subject, error) {
	acc, ok := s.accounts.Get(p.UserID)
	if !ok {
		return nil, nil
	}
	return []notify.Subject{
		notify.NewSmsSubject(acc.Phone, "", holder(acc)),           // SMS for the urgent alert
		notify.NewEmailSubject(acc.Email, "", "", "", holder(acc)), // email for the detailed report
	}, nil
}

func (s *TransactionService) otpSubjects(_ context.Context, p model.OtpPayload) ([]notify.Subject, error) {
	acc, ok := s.accounts.Get(p.UserID)
	if !ok {
		return nil, nil
	}
	return []notify.Subject{notify.NewSmsSubject(acc.Phone, "", holder(acc))}, nil
}

func (s *TransactionService) transferSubjects(_ context.Context, p model.TransactionPayload) ([]notify.Subject, error) {
	acc, ok := s.accounts.Get(p.FromAccountID)
	if !ok {
		return nil, nil
	}
	return []notify.Subject{notify.NewEmailSubject(acc.Email, "", "", "", holder(acc))}, nil
}

func (s *TransactionService) dailySummarySubjects(_ context.Context, p model.BalanceSummaryPayload) ([]notify.Subject, error) {
	acc, ok := s.accounts.Get(p.AccountID)
	if !ok {
		return nil, nil
	}
	return []notify.Subject{notify.NewEmailSubject(acc.Email, "", "", "", holder(acc))}, nil
}

// ── vocabulary supplier ──────────────────────────────────────────────────────

func (s *TransactionService) transferVocabulary(_ context.Context, p model.TransactionPayload) (any, error) {
	name := func(id string) string {
		if a, ok := s.accounts.Get(id); ok {
			return a.HolderName
		}
		return "unknown"
	}
	s.log.Info("Enriching transfer vocabulary", "sender", name(p.FromAccountID), "receiver", name(p.ToAccountID))
	return p, nil
}

// ── rules ────────────────────────────────────────────────────────────────────

func (s *TransactionService) fraudCheckRule(_ context.Context, p model.TransactionPayload) (bool, error) {
	passed := p.Amount <= 50000.0
	s.log.Info("🔍 Fraud check", "transaction", p.TransactionID, "passed", passed, "amount", p.Amount)
	return passed, nil
}

func (s *TransactionService) velocityCheckRule(_ context.Context, p model.TransactionPayload) (bool, error) {
	recent := s.transactions.CountByFromAccount(p.FromAccountID)
	passed := recent < 5
	s.log.Info("⚡ Velocity check", "account", p.FromAccountID, "passed", passed, "recent", recent)
	return passed, nil
}

// ── callbacks ────────────────────────────────────────────────────────────────

func (s *TransactionService) beforeSuspiciousLogin(_ context.Context, p model.LoginPayload) error {
	s.log.Warn("⏳ [BEFORE] Security alert processing", "user", p.UserID, "location", p.Location)
	return nil
}

func (s *TransactionService) afterLargeTransfer(_ context.Context, p model.TransactionPayload) error {
	s.log.Info("✅ [AFTER] Audit trail recorded", "transaction", p.TransactionID, "amount", p.Amount, "currency", p.Currency)
	return nil
}
