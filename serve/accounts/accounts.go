package accounts

import (
	"github.com/aboodman/replicant/serve"
)

var (
	a = []serve.Account{
		serve.Account{
			ID:     "0",
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
			Pubkey: []byte("-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEwNhpc2KRnxQRq2YETuKJShSC623E\nFFlAf4j2fFDKkEDM0Py9HCd3UDmxLyF7HUv9GC9mA+78HahKQnF8F37jLQ==\n-----END PUBLIC KEY-----"),
		},
	}
)

// Accounts returns a hardcoded set of currently known accounts.
func Accounts() []serve.Account {
	return a
}
