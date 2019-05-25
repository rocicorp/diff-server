package exec

const bootstrap = `
var commands = {
	'put': 0,
	'has': 1,
	'get': 2,
};

function recv(fn, args) {
	var f = this[fn];
	if (!f) {
		throw new Error('Unknown function: ' + fn);
	}

	args = JSON.parse(args);

	var handleError = function(res) {
		if (res.error) {
			throw new Error(res.error);
		}
		return res;
	};

	var db = {
		put: function(id, val) {
			handleError(send(commands.put, id, JSON.stringify(val)));
		},

		has: function(id) {
			return handleError(send(commands.has, id)).ok;
		}

		get: function(id) {
			var res = handleError(send(commands.get, id));
			return res.ok ? JSON.parse(res.data) : undefined;
		},
	};

	args.splice(0, 0, db);
	f.apply(null, args);
}
`
