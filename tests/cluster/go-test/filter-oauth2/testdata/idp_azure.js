module.exports.testcases = {
	"Azure AD": {
		resource: "https://ambassador.ambassador.svc.cluster.local/azure/httpbin/headers",
		username: "testuser@aprotesting.onmicrosoft.com",
		password: "6qak5GgDMgd/6iNFfuw5jA==",
	},
	"Azure AD (other domain)": {
		resource: "https://foo.ambassador.svc.cluster.local/azure/httpbin/headers",
		username: "testuser@aprotesting.onmicrosoft.com",
		password: "6qak5GgDMgd/6iNFfuw5jA==",
	},
};

module.exports.authenticate = async function(browsertab, username, password) {
	// page 1: Username
	await browsertab.waitForSelector('input[type="email"]', { visible: true });
	await browsertab.waitForSelector('input[type="submit"][value="Next"]', { visible: true });

	let domain_hint = (new URL(browsertab.url())).searchParams.get("domain_hint")
	if (domain_hint !== "aprotesting.onmicrosoft.com") {
		throw new Error(`domain_hint extraAuthorizationParameter not set properly: actual=${domain_hint} expected=aprotesting.onmicrosoft.com`);
	}

	await browsertab.type('input[type="email"]', username);
	await browsertab.click('input[type="submit"][value="Next"]');
	// page 2: Password
	await browsertab.waitForSelector('input[type="password"]', { visible: true });
	await browsertab.waitForSelector('input[type="submit"][value="Sign in"]', { visible: true });
	await browsertab.type('input[type="password"]', password);
	await browsertab.click('input[type="submit"][value="Sign in"]');
	// page 3: "Stay signed in?"
	await browsertab.waitForSelector('input[type="submit"][value="Yes"]', { visible: true });
	await browsertab.click('input[type="submit"][value="Yes"]');
};
