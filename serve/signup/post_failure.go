package signup

const PostFailureTemplateName = "post_failure"

// PostFailureTemplate is the HTML template rendered in response to an
// unsuccessful customer signup.
const PostFailureTemplate = `
<html>

<head>
    <title>Replicache Account Signup: Oops!</title>
</head>

<body>
    <h2>Could not create account :(</h2>

	<p>Your form submission had the following problem(s):
	<ul>
	{{range .Reasons}}
			<li><strong>{{.}}</strong></li>
	{{end}}
	</ul>

	<p>Please hit "back", correct the problem(s), and submit again. If you feel that you've reached this<br>
	page in error, our apologies, please email <a href="mailto:support@replicache.dev">support@replicache.dev</a>.<br>
	Thanks! 
 </body>

</html>
`
