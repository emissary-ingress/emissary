module.exports.testcases = {
	"Azure AD": {
		resource: "https://ambassador.standalone.svc.cluster.local/azure/httpbin/headers",
		username: "testuser@aprotesting.onmicrosoft.com",
		password: "VEcctB+mP9D8Me/kZyFQ",
	},
};

module.exports.authenticate = async function(browsertab, username, password) {
	// page 1: Username
	await browsertab.waitForSelector('input[type="email"]', { visible: true });
	await browsertab.waitForSelector('input[type="submit"][value="Next"]', { visible: true });
	await browsertab.type('input[type="email"]', username);
	await browsertab.click('input[type="submit"][value="Next"]');
	// page 2: Password
	await browsertab.waitForSelector('input[type="password"]', { visible: true });
	await browsertab.waitForSelector('input[type="submit"][value="Sign in"]', { visible: true });
	await browsertab.type('input[type="password"]', password);
	await browsertab.click('input[type="submit"][value="Sign in"]');
	// page 3: "Stay signed in?"
	await browsertab.waitForSelector('input[type="submit"][value="Yes"]', { visible: true });
	const done = browsertab.waitForNavigation();
	await browsertab.click('input[type="submit"][value="Yes"]');
	await done;
};
