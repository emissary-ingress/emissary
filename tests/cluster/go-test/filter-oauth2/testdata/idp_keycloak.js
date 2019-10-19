module.exports.testcases = {
	"Keycloak": {
		resource: "https://ambassador.default.svc.cluster.local/keycloak/httpbin/headers",
		username: "developer",
		password: "developer",
	},
};

module.exports.authenticate = async function(browsertab, username, password) {
	await browsertab.type('input#username', username);
	await browsertab.type('input#password', password);
	await browsertab.click('input#kc-login');
};
