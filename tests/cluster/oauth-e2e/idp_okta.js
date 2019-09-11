module.exports.testcases = {
	"Okta": {
		resource: "https://ambassador.standalone.svc.cluster.local/okta/httpbin/headers",
		username: "testificate+000@datawire.io",
		password: "Qwerty123",
	},
};

module.exports.authenticate = async function(browsertab, username, password) {
	await browsertab.type('#okta-signin-username', username);
	await browsertab.type('#okta-signin-password', password);
	await browsertab.click('#okta-signin-submit');
};
