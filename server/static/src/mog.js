// @flow

var Actions = Reflux.createActions([
	'active', // active song
	'error',
	'playlist',
	'protocols',
	'status',
	'tracks',
]);

var Stores = {};

_.each(Actions, function(action, name) {
	Stores[name] = Reflux.createStore({
		init: function() {
			this.listenTo(action, this.update);
		},
		update: function(data) {
			this.data = data;
			this.trigger.apply(this, arguments);
		}
	});
});

var browserStream;
var sxhr;

func streamBrowser() {
	if (!browserStream) {
		if (sxhr) {
			sxhr.abort();
			sxhr = null;
			console.log("abort SB XHR")
		}
		console.log("stop SB");
		return;
	}
	if (!sxhr) {
		sxhr = new XMLHttpRequest()
		sxhr.open("GET", "/api/stream", true);
		sxhr.responseType = "arraybuffer";
		sxhr.onload = function() {
			context.decodeAudioData(sxhr.response, function(buffer) {
				
			}
		};
		sxhr.send();
	}
}

browserStream = true;
streamBrowser();

function POST(path, params, success) {
	var data = new(FormData);
	if (_.isArray(params)) {
		_.each(params, function(v) {
			data.append(v.name, v.value);
		});
	} else if (_.isObject(params)) {
		_.each(params, function(v, k) {
			data.append(k, v);
		});
	} else if (params) {
		data = params;
	}
	var f = fetch(path, {
		method: 'post',
		body: data
	});
	f.then(function(response) {
		if (response.status >= 200 && response.status < 300) {
			return Promise.resolve(response);
		} else {
			return Promise.reject(new Error(response.statusText));
		}
	});
	f.catch(function(err) {
		alert(err);
	});
	if (success) {
		f.then(success);
	}
	return f;
}

function mkcmd(cmds) {
	return _.map(cmds, function(val) {
		return {
			"name": "c",
			"value": val
		};
	});
}

document.addEventListener('keydown', function(e) {
	if (document.activeElement != document.body) {
		return;
	}
	var cmd;
	switch (e.keyCode) {
	case 32: // space
		cmd = 'pause';
		break;
	case 37: // left
		cmd = 'prev';
		break;
	case 39: // right
		cmd = 'next';
		break;
	default:
		return;
	}
	POST('/api/cmd/' + cmd);
	e.preventDefault();
});

var Time = React.createClass({
	render: function() {
		var t = this.props.time / 1e9;
		var m = Math.floor(t / 60);
		var s = Math.floor(t % 60);
		if (s < 10) {
			s = "0" + s;
		}
		return <span>{m}:{s}</span>;
	}
});

function mkIcon(name) {
	return 'icon fa fa-border fa-lg clickable ' + name;
}
