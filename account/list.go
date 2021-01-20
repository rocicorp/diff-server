package account

var (
	RegularAccounts = []Record{
		{
			ID:              0,
			Name:            "Sandbox",
			ClientViewHosts: []string{"localhost"},
			ClientViewURLs:  []string{"http://localhost:8000/replicache-client-view"},
		},
		{
			ID:              1,
			Name:            "Replicache Sample TODO",
			ClientViewHosts: []string{"replicache-sample-todo.now.sh"},
			ClientViewURLs:  []string{"https://replicache-sample-todo.now.sh/serve/replicache-client-view"},
		},
		// Inactive
		// {
		// 	ID:            2,
		//  Name:          "Cron",
		//  ClientViewURLs: []string{"https://api.cron.app/replicache-client-view"},
		// },
		{
			ID:              3,
			Name:            "Songbook Studio",
			ClientViewHosts: []string{"us-central1-songbookstudio.cloudfunctions.net"},
			ClientViewURLs:  []string{"https://us-central1-songbookstudio.cloudfunctions.net/repliclient/4rzcWwvc83dlTz3CoX9WY8NHUxV2"},
		},
		{
			ID:              4,
			Name:            "Songbook Studio (Vercel)",
			ClientViewHosts: []string{"songbook.studio"},
			ClientViewURLs:  []string{"https://songbook.studio/api/repliclient"},
		},
	}
)
