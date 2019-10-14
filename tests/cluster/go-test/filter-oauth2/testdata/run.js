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
			console.log("final url is: "+browsertab.url());
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
				process.exit(99);
			}
		});
};

const sleep = function(ms) {
	return new Promise((resolve, reject) => {
		setTimeout(() => { resolve(); }, ms);
	});
};

const queueFrame = function(browsertab, shotdir) {
	let ts = Date.now();
	console.log("before frame", ts);
	return browsertab.screenshot({path: shotdir+"/"+ts+".png"})
		.then((screenshot) => {
			console.log("after frame", ts);
			return queueFrame(browsertab, shotdir);
		})
		.catch((err) => {
			console.log("screenshot error at "+ts+":", err);
		});
}

// This function is closely coupled with browser_test.go:browserTest().
const browserTest = function(timeout_ms, shotdir, fn) {
	resolveTestPromise(withBrowserTab(async (browsertab) => {
		browsertab.on('framenavigated', () => {
			console.log("framenavigated: "+browsertab.url());
		});

		queueFrame(browsertab, shotdir);

		try {
			await withTimeout(timeout_ms, fn(browsertab));
		} finally {
			await sleep(1000);
		}
	}));
};

module.exports.TestSkipError = TestSkipError;
module.exports.browserTest = browserTest;
