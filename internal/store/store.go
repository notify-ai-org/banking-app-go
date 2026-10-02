// Package store holds in-memory accounts and transactions.
package store

import (
	"sync"

	"github.com/notify-ai-org/banking-app-go/internal/model"
)

// AccountStore is an in-memory account store seeded with dummy data.
type AccountStore struct {
	mu       sync.RWMutex
	accounts map[string]model.Account
}

func NewAccountStore() *AccountStore {
	return &AccountStore{accounts: map[string]model.Account{
		"ACC-1": {ID: "ACC-1", HolderName: "David Park", Email: "rohan.nn1203@gmail.com", Phone: "+1-555-0201", Balance: 125000.50},
		"ACC-2": {ID: "ACC-2", HolderName: "Emily Chen", Email: "rohan.nn1203@gmail.com", Phone: "+1-555-0202", Balance: 45000.00},
		"ACC-3": {ID: "ACC-3", HolderName: "Frank Miller", Email: "rohan.nn1203@gmail.com", Phone: "+1-555-0203", Balance: 250000.75},
	}}
}

func (s *AccountStore) Get(id string) (model.Account, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.accounts[id]
	return a, ok
}

// TransactionStore tracks recent transactions for velocity checks.
type TransactionStore struct {
	mu           sync.RWMutex
	transactions map[string]model.TransactionPayload
}

func NewTransactionStore() *TransactionStore {
	return &TransactionStore{transactions: map[string]model.TransactionPayload{}}
}

func (s *TransactionStore) Save(tx model.TransactionPayload) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.transactions[tx.TransactionID] = tx
}

func (s *TransactionStore) Get(id string) (model.TransactionPayload, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tx, ok := s.transactions[id]
	return tx, ok
}

// CountByFromAccount counts transactions sent from an account.
func (s *TransactionStore) CountByFromAccount(accountID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, tx := range s.transactions {
		if tx.FromAccountID == accountID {
			n++
		}
	}
	return n
}
