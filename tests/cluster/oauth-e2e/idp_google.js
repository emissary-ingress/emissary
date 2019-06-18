var server = require('http').createServer((request, response) => {
	response.writeHead(307, {"Location": "https://ambassador.standalone.svc.cluster.local"+request.url});
	response.end();
});

module.exports.testcases = {
	"Google": {
		resource: "https://ambassador.standalone.svc.cluster.local/google/httpbin/headers",
		username: "ambassadorprotesting@gmail.com",
		password: "IN5Kji47teRW2bJMh39O",
		before: () => { server.listen(31001); },
		after: () => { server.close(); },
	},
};

module.exports.authenticate = async function(browsertab, username, password) {
	// page 1: Username
	await browsertab.waitForSelector('input[type="email"]', { visible: true });
	await browsertab.waitForSelector('[role="button"]#identifierNext', { visible: true });
	await browsertab.type('input[type="email"]', username);
	await browsertab.click('[role="button"]#identifierNext');
	// page 2: Password
	await browsertab.waitForSelector('input[type="password"]', { visible: true });
	await browsertab.waitForSelector('[role="button"]#passwordNext', { visible: true });
	await browsertab.type('input[type="password"]', password);
	const done = browsertab.waitForFunction(() => {
		// Google does several JS-based redirects, so
		// waitForNavigation and friends don't wait long
		// enough.
		return window.location.hostname == "ambassador.standalone.svc.cluster.local";
	});
	await browsertab.click('[role="button"]#passwordNext');
	await done;
};
