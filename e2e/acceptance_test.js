const dotenv = require('dotenv')
const puppeteer = require('puppeteer');
const { expect } = require('chai');

const result = dotenv.config({path: __dirname + "/env.in"})
if (result.error) {
  throw result.error
}

const env = result.parsed

beforeEach (async function () {
  process.on("unhandledRejection", (reason, p) => {
    console.error("Unhandled Rejection at: Promise", p, "reason:", reason);
  });

  global.browser = await puppeteer.launch({
    headless: true,
    timeout: 15000,
    devtools: false,
    ignoreHTTPSErrors: true,
    args: [
      '--disable-dev-shm-usage',
      '--no-sandbox'
    ]
  });
});

afterEach (function () {
  browser.close();
});

describe('user-agent', function () {  
  
  it('should be able to consent and see the content headers', async function () {
    const target = `http://${env.EXTERNAL_IP}/httpbin/headers`
    const page = await global.browser.newPage()
    const waitForNavagation = page.waitForNavigation({ waitUntil: "networkidle0" })
  
    await page.goto(target)
    await page.waitForSelector('input[type="email"]', { visible: true })
    await page.type('input[type="email"]', env.TESTUSER_EMAIL)
    await page.waitForSelector('input[type="password"]', { visible: true })
    await page.type('input[type="password"]', env.TESTUSER_PASSWORD)
    await page.waitForSelector('.auth0-lock-submit', { visible: true })
    await page.click('.auth0-lock-submit')
    await waitForNavagation

    const url = await page.evaluate(function () {
      return window.location.href
    })
    expect(url).to.be.equal(target)
    const cookies = await page.cookies()
    const cookie = cookies[0]

    expect(cookie, "access_token cookie not present").to.not.be.undefined
    expect(cookie.name).to.be.equal("access_token")
    expect(cookie.value.length, "access_token cookie value should be longer than 300 chars").to.be.above(300)
    expect(cookie.httpOnly, "access_token cookie should be http only").to.be.true

    const body = await page.evaluate(function () {
      return document.body.textContent
    })    
    expect(body, "page body not present").to.not.be.undefined
    
    const content = JSON.parse(body)
    expect(content.headers, "page content contain headers not present").to.not.be.undefined
    expect(content.headers.Authorization, "page content contain Authorization headers not present").to.not.be.undefined
  
  });

  it('should access ip without cookie', async function () {
    const target = `http://${env.EXTERNAL_IP}/httpbin/ip`
    const page = await global.browser.newPage()
    const waitForNavagation = page.waitForNavigation({ waitUntil: 'networkidle0' })
  
    await page.goto(target)
    await waitForNavagation

    const url = await page.evaluate(function () {
      return window.location.href
    })

    expect(url).to.be.equal(target)

    const cookies = await page.cookies()
    const cookie = cookies[0]

    expect(cookie, "access_token cookie not present").to.be.undefined
    
    const body = await page.evaluate(function () {
      return document.body.textContent
    })
    expect(body, "page body not present").to.not.be.undefined

    const content = JSON.parse(body)
    expect(content, "page content is undefined").to.not.be.undefined
    expect(content.origin, "page content contains origin").to.not.be.undefined
    expect(content.origin).to.not.be.equal(env.EXTERNAL_IP)
  });

});
