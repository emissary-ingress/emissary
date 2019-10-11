const fs = require('fs');
const puppeteer = require('puppeteer');

function TestSkipError(message) {
	this.name = "TestSkipError";
	this.message = "Test Skipped: " + message;
}
TestSkipError.prototype = Error.prototype;

const withBrowserTab = async function(fn) {
	const browser = await puppeteer.launch({
		//headless: false,
		ignoreHTTPSErrors: true,
		args: [
			'--disable-dev-shm-usage',
			'--no-sandbox'
		]
	});
	try {
		const browsertab = await browser.newPage();
		try {
			await fn(browsertab);
		} finally {
			console.log("url is: "+browsertab.url());
		}
	} finally {
		browser.close();
	}
};

const withTimeout = function(timeout_ms, promise) {
	return Promise.race([
		promise,
		new Promise(function(resolve, reject) {
			setTimeout(() => {
				reject('timed out after ' + timeout_ms + ' ms');
			}, timeout_ms);
		}),
	]);
};

const resolveTestPromise = function(promise) {
	promise.then(
		(value) => { process.exit(0); },
		(error) => {
			console.log(error);
			if (error instanceof TestSkipError) {
				process.exit(77);
			} else {
				process.exit(1);
			}
		});
};

const sleep = function(ms) {
	return new Promise((resolve, reject) => {
		setTimeout(() => { resolve(); }, ms);
	});
};

// This function is closely coupled with browser_test.go:browserTest().
const browserTest = function(timeout_ms, fn) {
	resolveTestPromise(withBrowserTab((browsertab) => {
		setInterval(() => {
			browsertab.screenshot().then((screenshot) => { fs.writeSync(3, screenshot); })
		}, 1000/5);

		return withTimeout(timeout_ms, fn(browsertab)).finally(() => { return sleep(1000); });
	}));
};

module.exports.TestSkipError = TestSkipError;
module.exports.browserTest = browserTest;
