// Package model holds the banking event payloads and account profile. The
// `notify` / `notifyDesc` struct tags are the Go form of the Java SDK's
// @Vocabulary annotation.
package model

// Account is the bank account profile used to resolve notification recipients.
type Account struct {
	ID         string  `json:"accountId" notify:"accountId" notifyDesc:"Unique bank account identifier"`
	HolderName string  `json:"holderName" notify:"holderName" notifyDesc:"Full name of the account holder"`
	Email      string  `json:"email" notify:"email" notifyDesc:"Account holder email address for banking notifications"`
	Phone      string  `json:"phone" notify:"phone" notifyDesc:"Account holder phone number for SMS banking notifications"`
	Balance    float64 `json:"balance" notify:"balance" notifyDesc:"Current available account balance"`
}

func (Account) NotifyModelDescription() string {
	return "Bank account profile used to resolve notification recipients and account context"
}

// LoginPayload is the payload for suspicious login detection events.
type LoginPayload struct {
	UserID    string `json:"userId" notify:"userId" notifyDesc:"User/account ID of the login attempt"`
	IPAddress string `json:"ipAddress" notify:"ipAddress" notifyDesc:"IP address of the login attempt"`
	DeviceID  string `json:"deviceId" notify:"deviceId" notifyDesc:"Device fingerprint or identifier"`
	Location  string `json:"location" notify:"location" notifyDesc:"Geo-location of the login attempt"`
	Timestamp string `json:"timestamp" notify:"timestamp" notifyDesc:"Timestamp of the login attempt (ISO-8601)"`
}

func (LoginPayload) NotifyModelDescription() string {
	return "Payload for suspicious login detection events"
}

// OtpPayload is the payload for OTP request events.
type OtpPayload struct {
	UserID    string `json:"userId" notify:"userId" notifyDesc:"User requesting the OTP"`
	OtpCode   string `json:"otpCode" notify:"otpCode" notifyDesc:"Generated OTP code"`
	Channel   string `json:"channel" notify:"channel" notifyDesc:"Delivery channel (SMS, EMAIL)"`
	ExpiresAt string `json:"expiresAt" notify:"expiresAt" notifyDesc:"OTP expiry timestamp (ISO-8601)"`
}

func (OtpPayload) NotifyModelDescription() string { return "Payload for OTP request events" }

// TransactionPayload is the payload for fund transfer events.
type TransactionPayload struct {
	TransactionID string  `json:"transactionId" notify:"transactionId" notifyDesc:"Unique transaction identifier"`
	FromAccountID string  `json:"fromAccountId" notify:"fromAccountId" notifyDesc:"Source account for the transfer"`
	ToAccountID   string  `json:"toAccountId" notify:"toAccountId" notifyDesc:"Destination account for the transfer"`
	Amount        float64 `json:"amount" notify:"amount" notifyDesc:"Transfer amount"`
	Currency      string  `json:"currency" notify:"currency" notifyDesc:"Currency code (e.g. USD, EUR)"`
	Type          string  `json:"type" notify:"type" notifyDesc:"Transaction type (WIRE, ACH, INTERNAL)"`
}

func (TransactionPayload) NotifyModelDescription() string { return "Payload for fund transfer events" }

// BalanceSummaryPayload is the payload for daily balance summary digest events.
type BalanceSummaryPayload struct {
	AccountID     string  `json:"accountId" notify:"accountId" notifyDesc:"Account for the balance summary"`
	Balance       float64 `json:"balance" notify:"balance" notifyDesc:"Current account balance"`
	Currency      string  `json:"currency" notify:"currency" notifyDesc:"Currency code"`
	StatementDate string  `json:"statementDate" notify:"statementDate" notifyDesc:"Statement date (YYYY-MM-DD)"`
}

func (BalanceSummaryPayload) NotifyModelDescription() string {
	return "Payload for daily balance summary digest events"
}
