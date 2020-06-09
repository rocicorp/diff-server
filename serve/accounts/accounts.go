package accounts

import (
	"roci.dev/diff-server/serve"
)

var (
	a = []serve.Account{
		serve.Account{
			ID:   "sandbox",
			Name: "Sandbox",
		},
		serve.Account{
			ID:            "1",
			Name:          "Replicache Sample TODO",
			ClientViewURL: "https://replicache-sample-todo.now.sh/serve/replicache-client-view",
		},
		serve.Account{
			ID:            "2",
			Name:          "Cron",
			ClientViewURL: "https://api.cron.app/replicache-client-view",
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
