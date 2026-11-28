// Optional headless UI acceptance against a retained, isolated real-provider run.
// PUPPETEER_MODULE, JSQR_MODULE and CHROME_BINARY select local QA tools.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const puppeteer = require(process.env.PUPPETEER_MODULE || 'puppeteer-core');
const folder = path.resolve(process.argv[2]);
const fixture = JSON.parse(fs.readFileSync(path.join(folder, 'trace-browser-private.json'), 'utf8'));
const output = path.join(folder, 'browser-evidence');
fs.mkdirSync(output, { recursive: true });

(async () => {
  const browser = await puppeteer.launch({ executablePath: process.env.CHROME_BINARY,
    headless: true, acceptInsecureCerts: true, userDataDir: path.join(folder, 'chrome-qa-profile') });
  try {
    const page = await browser.newPage();
    await page.setViewport({ width: 1440, height: 1080 });
    await browser.setCookie({ name: 'OMAMA_SESSION', value: fixture.customer_session, domain: '127.0.0.1', path: '/', httpOnly: true });
    await page.goto(fixture.web + '/orders/' + fixture.second_order_id, { waitUntil: 'domcontentloaded' });
    await page.waitForFunction(() => document.querySelector('#nguon-goc')?.innerText.includes('Hồ sơ hiện đã dừng công khai.'), { timeout: 90000 });
    const panel = await page.$('#nguon-goc');
    assert.equal(await panel.$$eval('a', links => links.length), 2);
    assert.equal(await panel.$$eval('p', rows => rows.filter(row => / · 1 sản phẩm/.test(row.textContent)).length), 2);
    await panel.screenshot({ path: path.join(output, 'order-two-sealed-batches.png') });
    await page.setViewport({ width: 390, height: 844 });
    await page.goto(fixture.qr_url, { waitUntil: 'domcontentloaded' });
    await page.waitForFunction(() => document.body.innerText.includes('Isolated traceability product'), { timeout: 60000 });
    assert.equal(await page.evaluate(() => document.body.innerText.includes('UNPUBLISHED CONTENT MUST NOT APPEAR')), false);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), true);
    await page.screenshot({ path: path.join(output, 'public-source-mobile.png'), fullPage: true });
    await page.goto(fixture.gone_qr_url, { waitUntil: 'domcontentloaded' });
    await page.waitForFunction(() => document.body.innerText.includes('Hồ sơ hiện không còn được công bố'), { timeout: 60000 });
    await page.screenshot({ path: path.join(output, 'public-source-gone-mobile.png'), fullPage: true });
    await browser.setCookie({ name: 'OMAMA_SESSION', value: fixture.staff_session, domain: '127.0.0.1', path: '/', httpOnly: true });
    await page.setViewport({ width: 1440, height: 1080 });
    await page.goto(fixture.web + '/admin/traceability?id=' + fixture.profile_id, { waitUntil: 'domcontentloaded' });
    async function click(text) {
      await page.waitForFunction(value => Array.from(document.querySelectorAll('button')).some(button => button.textContent === value && !button.disabled), { timeout: 60000 }, text);
      await page.evaluate(value => Array.from(document.querySelectorAll('button')).find(button => button.textContent === value).click(), text);
    }
    await click('Kiểm tra và chuẩn bị tem');
    await page.waitForSelector('.trace-print-label:not([hidden]) canvas', { timeout: 60000 });
    await page.addScriptTag({ path: process.env.JSQR_MODULE || require.resolve('jsqr') });
    const decodeCanvas = () => {
      const canvas = document.querySelector('.trace-print-label:not([hidden]) canvas');
      return window.jsQR(canvas.getContext('2d').getImageData(0, 0, canvas.width, canvas.height).data, canvas.width, canvas.height)?.data;
    };
    assert.equal(await page.evaluate(decodeCanvas), fixture.qr_url);
    assert.equal(await page.$eval('.trace-print-label:not([hidden])', element => element.innerText.includes('UNPUBLISHED CONTENT MUST NOT APPEAR')), false);
    const client = await page.createCDPSession();
    await client.send('Browser.setDownloadBehavior', { behavior: 'allow', downloadPath: output });
    for (const [label, extension] of [['Tải mã PNG', '.png'], ['Tải mã SVG', '.svg']]) {
      await click(label);
      await page.waitForFunction(() => !Array.from(document.querySelectorAll('button')).some(button => button.textContent === 'Tải mã PNG' && button.disabled));
      const deadline = Date.now() + 10000;
      while (!fs.readdirSync(output).some(name => name.startsWith('ma-nguon-goc-') && name.endsWith(extension)) && Date.now() < deadline) await new Promise(resolve => setTimeout(resolve, 100));
      const name = fs.readdirSync(output).find(name => name.startsWith('ma-nguon-goc-') && name.endsWith(extension));
      assert.ok(name, 'QR download did not finish');
      const data = fs.readFileSync(path.join(output, name));
      assert.ok(data.length > 100);
      if (extension === '.svg') assert.match(data.toString('utf8'), /<svg/);
      else {
        const decoded = await page.evaluate(async uri => {
          const image = new Image(); image.src = uri; await image.decode();
          const canvas = document.createElement('canvas'); canvas.width = image.width; canvas.height = image.height;
          const context = canvas.getContext('2d'); context.drawImage(image, 0, 0);
          return window.jsQR(context.getImageData(0, 0, canvas.width, canvas.height).data, canvas.width, canvas.height)?.data;
        }, 'data:image/png;base64,' + data.toString('base64'));
        assert.equal(decoded, fixture.qr_url);
      }
    }
    await page.evaluate(() => { window.print = () => { window.__fixturePrintInvoked = true; }; });
    await click('In tem');
    await page.waitForFunction(() => window.__fixturePrintInvoked === true);
    await page.emulateMediaType('print');
    await page.waitForSelector('body > .trace-print-sheet', { visible: true });
    await page.pdf({ path: path.join(output, 'source-label-print.pdf'), format: 'A4', printBackground: true });
    await (await page.$('body > .trace-print-sheet')).screenshot({ path: path.join(output, 'source-label-print.png') });
    const result = { status: 'PASS', real_order_two_batches: true, public_mobile_no_overflow: true,
      gone_mobile: true, draft_not_public: true, canvas_qr_decoded: true, downloaded_png_decoded: true,
      svg_download: true, print_portal_rendered: true, physical_printer: 'NOT RUN', physical_camera: 'NOT RUN' };
    fs.writeFileSync(path.join(output, 'result.json'), JSON.stringify(result, null, 2));
    console.log(JSON.stringify(result));
  } finally { await browser.close(); }
})().catch(error => { console.error(error.name + ': ' + error.message); process.exitCode = 1; });
