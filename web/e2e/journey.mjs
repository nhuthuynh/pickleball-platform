// The MVP journey, driven in a real browser: sign in -> create a profile ->
// add a facility and a court -> host a game -> find it -> join it.
//
// WHY THIS EXISTS. Every one of the 727 tests in `web/src` mounts a
// component with a fixture client. None of them opens a browser, so none
// could see the two defects that made this product unusable for 66 sprints:
// the app could not reach its own backend (#339, a browser's CORS rules),
// and a signed-in player had no way to get an account (#340, a missing
// screen). Both were found within half an hour of running the app and
// neither was findable any other way.
//
// It is deliberately NOT part of `make ci-checks`. It needs a Postgres, a
// running server and a built client — the three things CI does not have
// standing up — and a flaky gate that half the team cannot run is worse
// than an honest manual check. Run it by hand after a change to the client,
// the gateway, or either front door:
//
//   1. a Postgres with db/migrations applied
//   2. AUTH_ISSUER/AUTH_AUDIENCE/AUTH_JWKS_FILE from dev/auth, then
//      `go run ./cmd/server`
//   3. `npm --prefix web run dev`  (or the compose stack)
//   4. TOKEN=$(make -s dev-token) and pass it in a file
//
//   node web/e2e/journey.mjs <token-file> <screenshot-dir> <app-url>
//
// It asserts nothing by itself — it prints every /v1 call with its status
// and leaves screenshots behind. That is the point: the failures it catches
// are ones nobody predicted, so a passing assertion list would not have
// caught them either. Read the output.
//
// Known gap it does NOT cover: booking, payments, competitions and the
// share-link flows. One journey is not a suite, and pretending otherwise is
// how 724 green tests coexisted with an unusable product.

import { chromium } from 'playwright'
import { readFileSync } from 'node:fs'
const TOKEN = readFileSync(process.argv[2],'utf8').trim(), SHOTS=process.argv[3], APP=process.argv[4]
const log=(...a)=>console.log('[journey]',...a)
// Playwright resolves its own browser by default. PLAYWRIGHT_CHROMIUM_PATH
// overrides it for an environment that ships Chromium separately — this
// project's cloud containers pin one under /opt/pw-browsers that does not
// match the version playwright would download.
const browser = await chromium.launch({
  executablePath: process.env.PLAYWRIGHT_CHROMIUM_PATH || undefined,
  args: ['--no-sandbox'],
})
const page = await browser.newPage()
const net=[]
page.on('response', r=>{const u=new URL(r.url()); if(u.pathname.startsWith('/v1')) net.push(`${r.request().method().padEnd(5)} ${u.pathname.replace(/[0-9a-f-]{36}/g,'{id}')} -> ${r.status()}`)})
// only buttons INSIDE the step panel, so the wizard's tab strip is never clicked
const stepBtn = (re) => page.locator('.fo-step, section, form').locator('button:visible').filter({ hasText: re })
const dump = async (l) => log(`  [${l}]`, JSON.stringify(await page.locator('button:visible').evaluateAll(e=>e.map(x=>({t:x.textContent.trim().slice(0,26),d:x.disabled})))))

// --- 1. sign in and onboard -------------------------------------------
await page.goto(APP+'/sign-in',{waitUntil:'networkidle'})
await page.fill('#dev-token',TOKEN); await page.click('button[type="submit"]'); await page.waitForTimeout(300)
await page.goto(APP+'/welcome',{waitUntil:'networkidle'})
await page.fill('#profile-display-name','Nhu H'); await page.selectOption('#profile-level','3')
await page.click('[data-testid="create-profile"]'); await page.waitForTimeout(1200)
log('1. ONBOARDED, user id =', await page.evaluate(()=>localStorage.getItem('pickleball.authSession.userId')))

// --- 2. a facility and a court ----------------------------------------
await page.goto(APP+'/facilities/onboard',{waitUntil:'networkidle'})
await page.fill('#facility-name','Riverside Courts'); await page.fill('#facility-description','MVP venue'); await page.fill('#facility-address','5 Baseline Road')
await page.click('[data-testid="details-next"]'); await page.waitForTimeout(250)
await page.click('[data-testid="photos-next"]'); await page.waitForTimeout(1500)
await page.click('[data-testid="cameras-next"]'); await page.waitForTimeout(500)
await page.fill('#court-name-input','Court 1')
await page.click('button:has-text("Add court")'); await page.waitForTimeout(1200)
log('2. FACILITY + COURT created')

// --- 3. host a game ----------------------------------------------------
await page.goto(APP+'/games/new',{waitUntil:'networkidle'}); await page.waitForTimeout(600)
await page.click('button:has-text("Riverside Courts")')
await page.waitForSelector('input[type="checkbox"]', { timeout: 10000 })
await page.locator('input[type="checkbox"]').first().check()
await page.waitForTimeout(300)
for (let step=0; step<8; step++) {
  const publish = stepBtn(/^\s*Publish|Create game|Publish game/i).first()
  if (await publish.count() && await publish.isEnabled()) {
    log(`3.${step} PUBLISH -> "${(await publish.textContent()).trim()}"`)
    await publish.click(); await page.waitForTimeout(2500); break
  }
  const next = stepBtn(/Next:/).first()
  if (!(await next.count())) { await dump(`no-next-${step}`); break }
  if (!(await next.isEnabled())) {
    // datetime-local, not date+time — the first attempt filled neither and
    // the Next button stayed disabled with no error to explain why.
    const dts = page.locator('input[type="datetime-local"]:visible')
    for (let i=0;i<await dts.count();i++) {
      await dts.nth(i).fill(i===0 ? '2026-12-01T18:00' : '2026-12-01T20:00').catch(()=>{})
    }
    const nums = page.locator('input[type="number"]:visible')
    for (let i=0;i<await nums.count();i++) await nums.nth(i).fill('4').catch(()=>{})
    const radios=page.locator('input[type="radio"]:visible')
    if (await radios.count()) await radios.first().check().catch(()=>{})
    await page.waitForTimeout(400)
  }
  if (!(await next.isEnabled())) { await dump(`blocked-${step}`); await page.screenshot({path:`${SHOTS}/80-blocked-${step}.png`,fullPage:true}); break }
  log(`3.${step} ->`, (await next.textContent()).trim())
  await next.click(); await page.waitForTimeout(600)
}
await page.screenshot({path:SHOTS+'/81-after-publish.png',fullPage:true})
log('4. GAME PUBLISHED. alerts:', JSON.stringify(await page.locator('[role="alert"]:visible').evaluateAll(e=>e.map(a=>a.textContent.trim().slice(0,100)))))

// --- 4. find it on the games list and JOIN it -------------------------
await page.goto(APP+'/games',{waitUntil:'networkidle'}); await page.waitForTimeout(1200)
const listed = (await page.locator('body').innerText()).replace(/\s+/g,' ')
log('5. /games lists the new game (by its date):', /Dec 1, 2026/.test(listed))
await page.screenshot({path:SHOTS+'/82-games-list.png',fullPage:true})

// the list renders each game as a clickable card; the join panel lives
// inside the expanded one
const card = page.locator('button:visible').filter({ hasText: /2026/ }).first()
if (await card.count()) { await card.click(); await page.waitForTimeout(1200) }
log('5b. after opening the card:', (await page.locator('body').innerText()).replace(/\s+/g,' ').slice(0,300))
await page.screenshot({path:SHOTS+'/82b-card-open.png',fullPage:true})
const joinBtn = page.locator('button:visible').filter({ hasText: /^\s*(Join|Request to join|Join game|Confirm)/i }).first()
log('6. a Join control exists:', await joinBtn.count() > 0)
if (await joinBtn.count()) {
  await joinBtn.click()
  await page.waitForTimeout(2500)
  await page.screenshot({path:SHOTS+'/83-after-join.png',fullPage:true})
  const after = (await page.locator('body').innerText()).replace(/\s+/g,' ')
  log('7. page after joining:', after.slice(0, 260))
  log('8. alerts:', JSON.stringify(await page.locator('[role="alert"]:visible').evaluateAll(e=>e.map(a=>a.textContent.trim().slice(0,140)))))
} else {
  await page.screenshot({path:SHOTS+'/83-no-join.png',fullPage:true})
  log('7. buttons on /games:', JSON.stringify(await page.locator('button:visible').evaluateAll(e=>e.map(x=>x.textContent.trim().slice(0,30)))))
}
await browser.close()
log('--- calls ---'); net.forEach(l=>log('   ',l))
