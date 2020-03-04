const { expect } = require('chai');
const run = require('./run.js');

// can authorize requests
const standardTest = async (browsertab, idpfile, testname) => {
	if (idpfile.testcases[testname].before) {
		idpfile.testcases[testname].before();
	}
	try {
		const response = await browsertab.goto(idpfile.testcases[testname].resource);
		// verify that we got redirected to the IDP
		expect(response.request().redirectChain()).to.not.be.empty;
		expect((new URL(browsertab.url())).hostname).to.not.contain((new URL(idpfile.testcases[testname].resource)).hostname);
		// authenticate to the IDP
		let done = browsertab.waitForResponse(idpfile.testcases[testname].resource)
		    .then(() => browsertab.waitForFunction(() => {return document.readyState == "complete";}));
		await idpfile.authenticate(browsertab, idpfile.testcases[testname].username, idpfile.testcases[testname].password);
		await done;
		// verify that we got redirected properly
		expect(browsertab.url()).to.equal(idpfile.testcases[testname].resource);
		// verify that backend service received the authorization
		const echoedRequest = JSON.parse(await browsertab.evaluate(() => {return document.body.textContent}));
		expect(echoedRequest.headers.Authorization).to.match(/^Bearer /);
	} finally {
		if (idpfile.testcases[testname].after) {
			idpfile.testcases[testname].after();
		}
	}
};
module.exports.standardTest = standardTest;

// can be chained with other filters
module.exports.chainTest = async (browsertab, idpfile, testname) => {
	// this is mostly the same as the 'can authorize requests' test, but has more at the end
	const response = await browsertab.goto(idpfile.testcases[testname].resource);
	// verify that we got redirected to the IDP
	expect(response.request().redirectChain()).to.not.be.empty;
	expect((new URL(browsertab.url())).hostname).to.not.contain((new URL(idpfile.testcases[testname].resource)).hostname);
	// authenticate to the IDP
	let done = browsertab.waitForResponse(idpfile.testcases[testname].resource)
	    .then(() => browsertab.waitForFunction(() => {return document.readyState == "complete";}));
	await idpfile.authenticate(browsertab, idpfile.testcases[testname].username, idpfile.testcases[testname].password);
	await done;
	// verify that we got redirected properly
	expect(browsertab.url()).to.equal(idpfile.testcases[testname].resource);
	// verify that backend service received the authorization
	const echoedRequest = JSON.parse(await browsertab.evaluate(() => {return document.body.textContent}));
	expect(echoedRequest.headers.Authorization).to.match(/^Bearer /);

	// this is the extra bit at the end
	expect(echoedRequest.headers['X-Wikipedia']).to.not.be.undefined
};

// works with MS Office
const msofficeTest = async (browsertab, idpfile, testname, starturl) => {
	// this is mostly the same as the 'can authorize requests' test, but the starting URL is an argument
	const response = await browsertab.goto(starturl);
	// verify that we're alread at the IDP
	expect(response.request().redirectChain()).to.be.empty;
	expect((new URL(browsertab.url())).hostname).to.not.contain((new URL(idpfile.testcases[testname].resource)).hostname);
	// authenticate to the IDP
	let done = browsertab.waitForResponse(idpfile.testcases[testname].resource)
	    .then(() => browsertab.waitForFunction(() => {return document.readyState == "complete";}));
	await idpfile.authenticate(browsertab, idpfile.testcases[testname].username, idpfile.testcases[testname].password);
	await done;
	// verify that we got redirected properly
	expect(browsertab.url()).to.equal(idpfile.testcases[testname].resource);
	// verify that backend service received the authorization
	const echoedRequest = JSON.parse(await browsertab.evaluate(() => {return document.body.textContent}));
	expect(echoedRequest.headers.Authorization).to.match(/^Bearer /);
};
module.exports.msofficeTest = msofficeTest;

// can be turned off for specific paths
module.exports.disableTest = async (browsertab, idpfile, testname) => {
	const response = await browsertab.goto((new URL("ip", idpfile.testcases[testname].resource)).toString())
	// verify that there were no redirects
	expect(response.request().redirectChain()).to.be.empty;
	// verify that the response looks correct
	const responseBody = JSON.parse(await browsertab.evaluate(() => {return document.body.textContent}));
	expect(responseBody.origin).to.be.a('string');
};

const writeFile = (file, data) => {
	return new Promise((resolve, reject) => {
		fs.writeFile(file, data, error => {
			if (error) {
				reject(error);
			} else {
				resolve();
			}
		});
	});
};

module.exports.writeSessionID = async (browsertab, idpfile, testname, sessionFilename, xsrfFilename) => {
	await module.exports.standardTest(browsertab, idpfile, testname);
	console.log("now authenticated");
	let cookies = await browsertab.cookies();
	console.log("cookies:", cookies);
	let sessionID = cookies.find(c=>c.name.match(/^ambassador_session\./)).value;
	await writeFile(sessionFilename, sessionID);
	let xsrfToken = cookies.find(c=>c.name.match(/^ambassador_xsrf\./)).value;
	await writeFile(xsrfFilename, xsrfToken);
}
