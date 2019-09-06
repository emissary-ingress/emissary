const puppeteer = require('puppeteer');
const { expect } = require('chai');
const glob = require('glob');

const withBrowserTab = async function(fn) {
	const browser = await puppeteer.launch({
		headless: true,
		timeout: 5000,
		devtools: false,
		ignoreHTTPSErrors: true,
		args: [
			'--disable-dev-shm-usage',
			'--no-sandbox'
		]
	});
	try {
		const browsertab = await browser.newPage();
		await fn(browsertab);
	} finally {
		await browser.close();
	}
};

for (let idpFile of glob.sync("./idp_*.js")) {
	let idp = require(idpFile);
	for (const testname in idp.testcases) {
		let testcase = idp.testcases[testname];
		describe(testname, function() {
			it('can authorize requests', function() { return withBrowserTab(async (browsertab) => {
				if (testcase.before) {
					testcase.before();
				}
				try {
					const response = await browsertab.goto(testcase.resource);
					// verify that we got redirected to the IDP
					expect(response.request().redirectChain()).to.not.be.empty;
					expect((new URL(browsertab.url())).hostname).to.not.contain((new URL(testcase.resource)).hostname);
					// authenticate to the IDP
					let done = browsertab.waitForResponse(testcase.resource)
					    .then(() => browsertab.waitForFunction(() => {return document.readyState == "complete";}));
					await idp.authenticate(browsertab, testcase.username, testcase.password, () => {this.skip();});
					await done;
					// verify that we got redirected properly
					expect(browsertab.url()).to.equal(testcase.resource);
					// verify that backend service received the authorization
					const echoedRequest = JSON.parse(await browsertab.evaluate(() => {return document.body.textContent}));
					expect(echoedRequest.headers.Authorization).to.match(/^Bearer /);
				} finally {
					await browsertab.screenshot({path: idpFile.replace(/\.js$/, ".png")});
					if (testcase.after) {
						testcase.after();
					}
				}
			});});
			if (testname === "Auth0 (/httpbin)") {
				it('can be chained with other filters', function() { return withBrowserTab(async (browsertab) => {
					// this is mostly the same as the 'can authorize requests' test, but has more at the end

					const response = await browsertab.goto(testcase.resource);
					// verify that we got redirected to the IDP
					expect(response.request().redirectChain()).to.not.be.empty;
					expect((new URL(browsertab.url())).hostname).to.not.contain((new URL(testcase.resource)).hostname);
					// authenticate to the IDP
					let done = browsertab.waitForResponse(testcase.resource)
					    .then(() => browsertab.waitForFunction(() => {return document.readyState == "complete";}));
					await idp.authenticate(browsertab, testcase.username, testcase.password, () => {this.skip();});
					await done
					// verify that we got redirected properly
					expect(browsertab.url()).to.equal(testcase.resource);
					// verify that backend service received the authorization
					const echoedRequest = JSON.parse(await browsertab.evaluate(() => {return document.body.textContent}));
					expect(echoedRequest.headers.Authorization).to.match(/^Bearer /);

					// this is the extra bit at the end
					expect(echoedRequest.headers['X-Wikipedia']).to.not.be.undefined
				});});
				it('can be turned off for specific paths', function() { return withBrowserTab(async (browsertab) => {
					const response = await browsertab.goto((new URL("ip", testcase.resource)).toString())
					// verify that there were no redirects
					expect(response.request().redirectChain()).to.be.empty;
					// verify that the response looks correct
					const responseBody = JSON.parse(await browsertab.evaluate(() => {return document.body.textContent}));
					expect(responseBody.origin).to.be.a('string');
				});});
			}
		});
	}
}
