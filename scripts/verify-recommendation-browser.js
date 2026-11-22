// Run with playwright-cli run-code, after starting the isolated stack and Web.
async (page) => {
  await page.context().addCookies([
    { name: "OMAMA_SESSION", value: "f47ac10b-58cc-4372-a567-0e02b2c3d477", domain: "127.0.0.1", path: "/" },
    { name: "oma_rec_measurement", value: "denied", domain: "127.0.0.1", path: "/" }
  ]);
  await page.goto("http://127.0.0.1:13046/");
  const consent = page.getByRole("checkbox", { name: "Cá nhân hóa gợi ý bằng lịch sử xem, thêm giỏ và mua hàng của tôi." });
  await consent.waitFor();
  if (!(await consent.isChecked())) {
    const saved = page.waitForResponse(r => r.url().endsWith("/api/recommendations/consent") && r.request().method() === "PUT");
    await consent.click();
    if ((await saved).status() !== 200) throw new Error("Grant failed");
  }
  await page.goto("http://127.0.0.1:13046/products/product-4");
  const feedback = page.waitForResponse(r => r.url().endsWith("/api/recommendations/events") && r.request().postDataJSON()?.eventType === "VIEW_ITEM");
  await page.getByRole("region", { name: "Sản phẩm tương tự" }).waitFor();
  if ((await feedback).status() !== 202) throw new Error("View feedback was not accepted");
  await page.goto("http://127.0.0.1:13046/products/product-1");
  const related = page.getByRole("region", { name: "Sản phẩm tương tự" });
  await related.waitFor();
  if ((await related.getByRole("link").count()) !== 5) throw new Error("Expected five other products");
  await related.screenshot({ path: "output/playwright/recommendation-related.png" });
  await page.goto("http://127.0.0.1:13046/cart");
  const basket = page.getByRole("region", { name: "Thường mua cùng" });
  await basket.waitFor();
  if (!(await basket.innerText()).includes("Product 5")) throw new Error("Qualified pair missing");
  await basket.screenshot({ path: "output/playwright/recommendation-cart.png" });
  await page.goto("http://127.0.0.1:13046/");
  const home = page.getByRole("region", { name: "Dành cho bạn" });
  await home.waitFor();
  await home.screenshot({ path: "output/playwright/recommendation-home.png" });
  await page.evaluate(() => { window.__recEvents = []; window.gtag = (...args) => window.__recEvents.push(args); });
  await page.getByRole("checkbox", { name: "Cho phép đo hiệu quả gợi ý bằng Google Analytics." }).check();
  await home.scrollIntoViewIfNeeded();
  await page.waitForTimeout(1200);
  const impressions = await page.evaluate(() => window.__recEvents.filter(e => e[0] === "event" && e[1] === "view_item_list"));
  if (impressions.length !== 1) throw new Error("Expected one visible impression");
  await home.getByRole("link").first().click();
  const added = page.waitForResponse(r => r.url().endsWith("/api/cart/items") && r.request().method() === "POST");
  await page.getByRole("button", { name: /Thêm vào giỏ/ }).click();
  if ((await added).status() !== 200) throw new Error("Cart addition failed");
  await page.waitForFunction(() => window.__recEvents.some(e => e[0] === "event" && e[1] === "add_to_cart"));
  const names = await page.evaluate(() => window.__recEvents.filter(e => e[0] === "event").map(e => e[1]));
  if (!names.includes("select_item")) throw new Error("Click provenance missing");
  await page.goto("http://127.0.0.1:13046/");
  const checkbox = page.getByRole("checkbox", { name: "Cá nhân hóa gợi ý bằng lịch sử xem, thêm giỏ và mua hàng của tôi." });
  await checkbox.waitFor();
  const revoked = page.waitForResponse(r => r.url().endsWith("/api/recommendations/consent") && r.request().method() === "PUT");
  await checkbox.click();
  if ((await revoked).status() !== 200) throw new Error("Revoke failed");
  await page.getByRole("region", { name: "Sản phẩm nổi bật" }).waitFor();
  const result = await page.evaluate(async () => (await fetch("/api/recommendations?placement=home")).json());
  if (result.strategy === "PERSONALIZED") throw new Error("Personalization survived revoke");
  const csrf = await page.evaluate(async () => (await fetch("/api/recommendations/consent", {
    method: "PUT", headers: { "Content-Type": "application/json", "X-CSRF-Token": "invalid" },
    body: JSON.stringify({ enabled: true, policyVersion: "personalization-v1" })
  })).status);
  if (csrf !== 403) throw new Error("BFF accepted invalid CSRF");
  await page.route("**/api/recommendations?**", route => route.fulfill({ status: 503, contentType: "application/json", body: "{}" }));
  await page.goto("http://127.0.0.1:13046/products/product-1");
  await page.getByRole("heading", { name: "Product 1", exact: true }).waitFor();
  await page.waitForTimeout(1500);
  if (await page.getByRole("region", { name: "Sản phẩm tương tự" }).count()) throw new Error("Unavailable rail should be hidden");
  await page.unroute("**/api/recommendations?**");
  await page.context().addCookies([{ name: "oma_rec_measurement", value: "denied", domain: "127.0.0.1", path: "/" }]);
}
