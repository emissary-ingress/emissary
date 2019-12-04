module.exports.testcases = (() => {
	let testcases = {}
	let addTestcase = function(name) {
		testcases[`Auth0 (/${name})`] = {
			resource: `https://ambassador.standalone.svc.cluster.local/${name}/headers`,
			username: "testuser@datawire.com",
			password: "TestUser321",
		};
	};
	addTestcase('oauth2-auth0-nojwt-and-plugin-and-whitelist');
	addTestcase('oauth2-auth0-nojwt-and-k8ssecret-and-xhrerror');
	addTestcase('oauth2-auth0-nojwt-and-anyerror');
	addTestcase('oauth2-auth0-simplejwt');
	addTestcase('oauth2-auth0-complexjwt');
	return testcases;
})();

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
