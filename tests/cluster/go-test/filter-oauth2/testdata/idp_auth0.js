module.exports.testcases = {
	"Auth0 (/auth0/httpbin)": {
		resource: "https://ambassador.ambassador.svc.cluster.local/auth0/httpbin/headers",
		username: "testuser@datawire.com",
		password: "TestUser321",
	},
	"Auth0 (/auth0-k8s/httpbin)": {
		resource: "https://ambassador.ambassador.svc.cluster.local/auth0-k8s/httpbin/headers",
		username: "testuser@datawire.com",
		password: "TestUser321",
	},
	"Auth0 (/httpbin)": {
		resource: "https://ambassador.ambassador.svc.cluster.local/httpbin/headers",
		username: "testuser@datawire.com",
		password: "TestUser321",
	}
};

module.exports.authenticate = async function(browsertab, username, password) {
	console.log("[auth0] email...");
	await browsertab.waitForSelector('input[type="email"]', { visible: true });
	await browsertab.type('input[type="email"]', username);
	console.log("[auth0] password...");
	await browsertab.waitForSelector('input[type="password"]', { visible: true });
	await browsertab.type('input[type="password"]', password);
	console.log("[auth0] submit...");
	await browsertab.waitForSelector('.auth0-lock-submit', { visible: true });
	await browsertab.click('.auth0-lock-submit');
	console.log("[auth0] done");
};
