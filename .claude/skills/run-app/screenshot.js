// Screenshots of the running app (http://localhost:8080) in light/desktop and
// dark/phone, plus the settings dialog. Prints page errors per viewport.
// Usage: NODE_PATH=$(npm root -g) node screenshot.js <out-dir> [url]
const { chromium } = require("playwright");

(async () => {
  const out = process.argv[2] || ".";
  const url = process.argv[3] || "http://localhost:8080/";
  const browser = await chromium.launch();
  const views = [
    ["desktop-light", { width: 1280, height: 800 }, "light"],
    ["phone-dark", { width: 390, height: 844 }, "dark"],
  ];
  for (const [name, viewport, colorScheme] of views) {
    const page = await browser.newPage({ viewport, colorScheme });
    const errors = [];
    page.on("pageerror", (e) => errors.push(e.message));
    page.on("console", (m) => m.type() === "error" && errors.push(m.text()));
    await page.goto(url);
    await page.waitForTimeout(3500); // status poll + snapshot
    await page.screenshot({ path: `${out}/${name}.png`, fullPage: true });
    if (name === "desktop-light") {
      await page.click("#settingsBtn");
      await page.waitForTimeout(500);
      await page.screenshot({ path: `${out}/settings.png` });
    }
    console.log(name, errors.length ? errors : "no errors");
  }
  await browser.close();
})();
