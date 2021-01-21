package signup

const PostSuccessTemplateName = "post_success"

// PostSuccessTemplate is the HTML template rendered in response to a
// successful customer signup.
const PostSuccessTemplate = `
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
