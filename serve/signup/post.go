package signup

const PostTemplateName = "post"

// PostTemplate is the HTML template rendered in response to a
// customer signup.
const PostTemplate = `
<html>

<head>
    <title>Replicache Account Signup: Success!</title>
</head>

<body>
    <h2>Success!</h2>

    <h2>Your Account ID is {{ .ID }}</h2>
    <!-- TODO add some more text here: pass it Replicache constructor, etc. -->
</body>

</html>
`
