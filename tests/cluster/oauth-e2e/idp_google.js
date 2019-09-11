var server = require('http').createServer((request, response) => {
	response.writeHead(307, {"Location": "https://ambassador.standalone.svc.cluster.local"+request.url});
	response.end();
});

module.exports.testcases = {
	"Google": {
		resource: "https://ambassador.standalone.svc.cluster.local/google/httpbin/headers",
		username: "ambassadorprotesting@gmail.com",
		password: "NO2I27Bg1XY",
		before: () => { server.listen(31001); },
		after: () => { server.close(); },
	},
};

// This is a private variable instead of 'module.exports.authenticate' so we can wrap it below.
const authenticate = async function(browsertab, username, password) {
	// page 1: Username
	await browsertab.waitForSelector('input[type="email"]', { visible: true });
	await browsertab.waitForSelector('[role="button"]#identifierNext', { visible: true });
	await browsertab.type('input[type="email"]', username);
	await browsertab.click('[role="button"]#identifierNext');
	// page 2: Password
	await browsertab.waitForSelector('input[type="password"]', { visible: true });
	await browsertab.waitForSelector('[role="button"]#passwordNext', { visible: true });
	await browsertab.type('input[type="password"]', password);
	await browsertab.click('[role="button"]#passwordNext');
};

const waitUntilRender = function(browsertab) {
	return browsertab.waitForFunction(() => {
		let view = document.querySelector("#initialView");
		return (view === null) || (view.getAttribute("aria-busy") !== "true");
	});
};

module.exports.authenticate = function(browsertab, username, password, skipfn) {
	return Promise.race([
		// If at any point it decides to show us a Captcha, just skip the test :-/
		browsertab.waitForSelector('img#captchaimg', { visible: true })
			.then(() => {return waitUntilRender(browsertab);})
			.then(() => {skipfn();}),
		// If Google decides to reject the signin, just skip the test :(
		browsertab.waitForFunction(() => {return window.location.href.startsWith("https://accounts.google.com/signin/oauth/deniedsigninrejected?");})
			.then(() => {return waitUntilRender(browsertab);})
			.then(() => {skipfn();}),
		// otherwise, authenticate as normal.
		authenticate(browsertab, username, password),
	]);
};
