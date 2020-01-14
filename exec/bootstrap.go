package exec

const bootstrap = `
var commands = {
	'put': 0,
	'has': 1,
	'get': 2,
	'del': 3,
	'scan': 4,
};

var db = (function() {
	var handleError = function(res) {
		if (res.error) {
			throw new Error(res.error);
		}
		return res;
	};

	var validID = function(id) {
		if (!id) {
			throw new Error("Invalid id");
		}
	};

	return {
		put: function(id, val) {
			validID(id);
			var undef;
			if (val === null || val === undef) {
				throw new Error("Invalid value");
			}
			handleError(send(commands.put, id, JSON.stringify(val)));
		},

		has: function(id) {
			validID(id);
			return handleError(send(commands.has, id)).ok;
		}

		get: function(id) {
			validID(id);
			var res = handleError(send(commands.get, id));
			return res.ok ? JSON.parse(res.data) : undefined;
		},

		scan: function(opts) {
			if (!opts) {
				throw new Error('Invalid param');
			}
			var res = handleError(send(commands.scan, JSON.stringify(opts)));
			return res.ok ? JSON.parse(res.data) : undefined;
		},

		del: function(id) {
			validID(id);
			var res = handleError(send(commands.del, id));
			return res.ok;
		},
	};
})();

function recv(fn, args) {
	var f = this[fn];
	if (!f) {
		return {ok: false};
	}
	var parsed = JSON.parse(args);
	var res = f.apply(null, parsed);
	return {
		ok: true,
		result: res === undefined ? res : JSON.stringify(res),
	};
}
`
