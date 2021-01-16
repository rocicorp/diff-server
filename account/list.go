package account

var (
	RegularAccounts = []Record{
		{
			ID:             0,
			Name:           "Sandbox",
			ClientViewURLs: []string{"http://replicache.dev"},
		},
		{
			ID:             1,
			Name:           "Replicache Sample TODO",
			ClientViewURLs: []string{"https://replicache-sample-todo.now.sh/serve/replicache-client-view"},
		},
		// Inactive
		// {
		// 	ID:            2,
		//  Name:          "Cron",
		//  ClientViewURLs: []string{"https://api.cron.app/replicache-client-view"},
		// },
		{
			ID:             3,
			Name:           "Songbook Studio",
			ClientViewURLs: []string{"https://us-central1-songbookstudio.cloudfunctions.net/repliclient/4rzcWwvc83dlTz3CoX9WY8NHUxV2"},
		},
		{
			ID:             4,
			Name:           "Songbook Studio (Vercel)",
			ClientViewURLs: []string{"https://songbook.studio/api/repliclient"},
		},
	}
)
