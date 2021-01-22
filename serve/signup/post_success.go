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

	Please note:
	<ul>
	  <li>Your account is suitable for <big><big><strong>evaluation purposes</strong></big></big>. To deploy to<br>
	  end users in production for non-evaluation purposes, please contact us at <a href="mailto:support@replicache.dev">support@replicache.dev</a>.<br>
	  (We just need to lift some default limits for you and ensure you agree to our <a href="https://github.com/rocicorp/repc/blob/main/licenses/BSL.txt">BSL license</a>.)<br><br>

	  <li>You need to include your Account ID in the <i>diffServerAuth</i> field when<br>
	  you <a href="https://github.com/rocicorp/replicache-sdk-js#%EF%B8%8F-instantiate">instantiate Replicache</a>
	  in your JavaScript application.

	</ul>

	<p>Potential next steps: <a href="https://github.com/rocicorp/replicache/blob/main/README.md">Replicache README</a>,
	<a href="https://js.replicache.dev/#replicache-js-sdk">Replicache JS Quick start</a>, or
	<a href="mailto:support@replicache.dev">contact us at support@replicache.dev</a>.
</body>

</html>
`
