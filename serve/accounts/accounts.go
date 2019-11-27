package accounts

import (
	"roci.dev/replicant/serve"
)

var (
	a = []serve.Account{
		serve.Account{
			ID:     "sandbox",
			Name:   "Sandbox",
			Pubkey: nil,
		},
		serve.Account{
			ID:     "1",
			Name:   "Rocicorp",
			Pubkey: []byte("-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE4DwXA3SHZ7TpzahAgOTRRgblBGxL\ndOHVmZ/J1bgBuuxMZzkassAsUCFCaMNu5HZuFUh98kA1laxZzs78O9EDQw==\n-----END PUBLIC KEY-----"),
		},
		serve.Account{
			ID:     "2",
			Name:   "Turtle Technologies, Inc.",
			Pubkey: nil,
		},
	}
)

// Accounts returns a hardcoded set of currently known accounts.
func Accounts() []serve.Account {
	return a
}
