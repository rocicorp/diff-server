package signup

const GetTemplateName = "get"

// GetTemplate is the HTML template for the page a customer uses to sign up.
// The template is included statically because I cannot figure out how
// to get Vercel to include template files so they are accessible at function
// startup time. (I tried both "functions" with includeFiles and static
// build rules.)
const GetTemplate = `
<!doctype html>
<html>

<head>
    <title>Replicache Account Signup</title>
</head>

<body>
    <h2>Replicache Account Signup</h2>

    <p>Please fill out the form below to generate an Account ID for Replicache.<br>
    You need to include your Account ID in the <i>diffServerAuth</i> field when<br>
    you <a href="https://github.com/rocicorp/replicache-sdk-js#%EF%B8%8F-instantiate">instantiate Replicache</a>
    in your JavaScript application.

	<p>Note that your account will be suitable for <big><big><strong>evaluation purposes</strong></big></big>, but<br>
	due to licensing and default account limitations, it will not be suitable for<br>
	deployment to end users in production "for real." In order to deploy Replicache<br>
	to end users in production for non-evaluation purposes, please email us<br>
	at <a href="mailto:support@replicache.dev">support@replicache.dev</a> to upgrade your account.<br><br>

    <form method="POST" action="/signup">
        <label>Name: <input type="text" name="{{.Name}}"></label><br>
        <label>Email: <input type="text" name="{{.Email}}"></label><br>
        <input type="submit">
    </form>

</body>

</html>
`
