const run = require('./run.js');

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

const clickNext = async function(browsertab) {
	const button = await Promise.race([
		browsertab.waitForSelector('[role="button"]#identifierNext', { visible: true }),
		browsertab.waitForSelector('[role="button"]#passwordNext', { visible: true }),
		browsertab.waitForSelector('.rc-button-submit', { visible: true }),
	]);
	await button.click();
};

// This is a private variable instead of 'module.exports.authenticate' so we can wrap it below.
const authenticate = async function(browsertab, username, password) {
	// page 1: Username
	await browsertab.waitForSelector('input[type="email"]', { visible: true });
	await browsertab.waitForFunction(() => {return document.activeElement === document.querySelector('input[type="email"]');});
	await browsertab.type('input[type="email"]', username);
	await clickNext(browsertab);
	// page 2: Password
	await browsertab.waitForSelector('input[type="password"]', { visible: true });
	await browsertab.type('input[type="password"]', password);
	await clickNext(browsertab);

	await browsertab.waitForResponse((resp) => {return resp.url().startsWith("http://localhost:31001/callback?");})
};

const waitUntilRender = function(browsertab) {
	return browsertab.waitForFunction(() => {
		let view = document.querySelector("#initialView");
		return (view === null) || (view.getAttribute("aria-busy") !== "true");
	});
};

const handleChallenges = async function(browsertab) {
	await Promise.race([
		// Confirm recovery email (old?)
		browsertab.waitForFunction(() => {return window.location.href.startsWith("https://accounts.google.com/signin/v2/challenge/selection?");}).then(async () => {
			await waitUntilRender(browsertab);

			await browsertab.waitForSelector('[role="link"][data-challengetype="12"]', { visible: true });
			await browsertab.click('[role="link"][data-challengetype="12"]');
			await browsertab.click('[role="link"][data-challengetype="12"]'); // IDK why, try clicking it twice

			await browsertab.waitForSelector('input[type="email"]', { visible: true });
			await browsertab.waitForSelector('[role="button"]', { visible: true });
			await browsertab.type('input[type="email"]', "dev+apro-gmail@datawire.io");
			await clickNext(browsertab);
		}),
		// Confirm recovery email (new?)
		browsertab.waitForFunction(() => {return window.location.href.startsWith("https://accounts.google.com/signin/selectchallenge/");}).then(async () => {
			await waitUntilRender(browsertab);

			let inputChoose = await browsertab.waitForSelector('form[action="/signin/challenge/kpe/4"] button[type="submit"]', { visible: true });
			await inputChoose.click();

			let inputEmail = await browsertab.waitForSelector('input[type="email"]', { visible: true });
			let inputDone = await browsertab.waitForSelector('input[type="submit"][value="Done"]', { visible: true });
			await inputEmail.type("dev+apro-gmail@datawire.io");
			await inputDone.click();
		}),
		// Click "confirm" to confirm the recovery info
		browsertab.waitForFunction(() => {return window.location.href.startsWith("https://myaccount.google.com/signinoptions/recovery-options-collection?");}).then(async () => {
			let button = await browsertab.waitForXPath('//div[@role="button"]//span[contains(text(), "Confirm")]');
			await button.click();
		}),
		// Click "next" for transient errors
		browsertab.waitForFunction(() => {return window.location.href.startsWith("https://accounts.google.com/info/unknownerror?");}).then(async () => {
			await browsertab.waitForSelector('[role="button"]', { visible: true });
			await browsertab.click('[role="button"]');
		}),
		// Click "allow" for consent-confirmation
		browsertab.waitForFunction(() => {return window.location.href.startsWith("https://accounts.google.com/signin/oauth/legacy/consent?");}).then(async () => {
			await browsertab.waitForSelector('button#submit_approve_access', { visible: true });
			await browsertab.click('button#submit_approve_access');
		}),
	]);
	return handleChallenges(browsertab);
};

module.exports.authenticate = function(browsertab, username, password) {
	return Promise.race([
		// If at any point it decides to show us a Captcha, just skip the test :-/
		browsertab.waitForSelector('img#captchaimg', { visible: true })
			.then(() => {return waitUntilRender(browsertab);})
			.then(() => {return Promise.reject(new run.TestSkipError("captcha"));}),
		// If Google decides to reject the signin, just skip the test :(
		browsertab.waitForFunction(() => {return window.location.href.startsWith("https://accounts.google.com/signin/oauth/deniedsigninrejected?");})
			.then(() => {return waitUntilRender(browsertab);})
			.then(() => {return Promise.reject(new run.TestSkipError("denied"));}),
		browsertab.waitForFunction(() => {return window.location.href.startsWith("https://accounts.google.com/signin/rejected?");})
			.then(() => {return waitUntilRender(browsertab);})
			.then(() => {return Promise.reject(new run.TestSkipError("denied"));}),
		// otherwise, authenticate as normal.
		authenticate(browsertab, username, password),
		// Click "next" and such if it decides to add extra pages; this recurses and  will never resolve
		handleChallenges(browsertab),
	]);
};
