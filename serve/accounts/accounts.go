package accounts

import (
	"roci.dev/diff-server/serve"
)

var (
	a = []serve.Account{
		serve.Account{
			ID:     "sandbox",
			Name:   "Sandbox",
			Pubkey: nil,
			// ClientViewURL: "<url>",
		},
		serve.Account{
			ID:     "1",
			Name:   "Rocicorp",
			Pubkey: []byte("-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE4DwXA3SHZ7TpzahAgOTRRgblBGxL\ndOHVmZ/J1bgBuuxMZzkassAsUCFCaMNu5HZuFUh98kA1laxZzs78O9EDQw==\n-----END PUBLIC KEY-----"),
			// ClientViewURL: "<url>",
		},
		serve.Account{
			ID:     "2",
			Name:   "Turtle Technologies, Inc.",
			Pubkey: []byte("-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEwNhpc2KRnxQRq2YETuKJShSC623E\nFFlAf4j2fFDKkEDM0Py9HCd3UDmxLyF7HUv9GC9mA+78HahKQnF8F37jLQ==\n-----END PUBLIC KEY-----"),
			// ClientViewURL: "<url>",
		},
	}
)

// Accounts returns a hardcoded set of currently known accounts.
func Accounts() []serve.Account {
	return a
}
