module.exports.testcases = {
	"UAA": {
		resource: "https://ambassador.standalone.svc.cluster.local/uaa/httpbin/headers",
		username: "testuser@example.com",
		password: "12345",
	},
};

module.exports.authenticate = async function(browsertab, username, password) {
	// page 1: authenticate
	await browsertab.type('input[name="username"]', username);
	await browsertab.type('input[name="password"]', password);
	const done = browsertab.waitForNavigation();
	await browsertab.click('input[type="submit"]');
	await done;
	// page 2: authorize (which it only sometimes shows)
	if ((new URL(browsertab.url())).hostname == "uaa.standalone.svc.cluster.local") {
		const done2 = browsertab.waitForNavigation();
		await browsertab.click('button#authorize');
		await done2;
	}
};
