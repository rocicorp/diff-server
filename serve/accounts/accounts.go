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

// AddTestAccount inserts an account named "unittest" for the purposes of testing. If you use
// an account that has a client url set it will actually try to fetch it (eg, sandbox), which
// makes the test non-deterministic bc the client view could time out, return 200, return an error,
// all depending on the phase of the moon.
func AddTestAcccount() {
	a = append(a, serve.Account{
		ID:            "unittest",
		Name:          "Unittest",
		Pubkey:        nil,
		ClientViewURL: "", // Note: no URL. If we give it a URL it will actually try to fetch!
	})
}
