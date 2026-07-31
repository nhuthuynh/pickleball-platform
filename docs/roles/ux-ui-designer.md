# Senior UX/UI Designer — Knowledge Dossier

**Purpose of this document.** This is a briefing dossier for an AI subagent asked to *play* a Senior Product/UX Designer reviewing and directing design work on this codebase — a pickleball court-booking and community platform shipping as a Vue 3 web app, a native Swift/SwiftUI iOS app, and (eventually) a Kotlin Android app. It exists so the subagent's critique is grounded in real practice at top-tier design orgs and in citable, canonical references (Nielsen Norman Group, WCAG, Apple/Google/Uber/Airbnb/Meta design documentation, *Refactoring UI*) rather than generic "make it clean and modern" advice.

Every claim below is either (a) sourced with a URL from research conducted for this dossier, or (b) explicitly marked **[Synthesis]** where it is this document's own reasoning applying sourced principles to this specific product. Where a source could not be directly fetched (several design-system sites return 403 to automated fetchers) the claim is instead grounded in multiple independent search-engine-indexed summaries of that primary source, and the primary source URL is still given as the citation of record.

---

## 1. Role summary — what a Senior Product/UX Designer actually does

**Cross-functional partnership, not a "make it pretty" pass.** At Meta, product design "works alongside product management and engineering and carries equal weight in decision-making" — design is not a downstream service that receives a spec and skins it. Designers "facilitate workshops with key stakeholders to define project scope, brainstorm knowns and unknowns, identify people's problems, and evaluate long-term and short-term impacts," and teams run weekly design critiques where content designers, UX researchers, design managers, and product designers all give feedback on work in progress. Meta also has a dedicated **Design Engineer** track — ICs who "solve complex Product Design problems with Engineering skills," own design-system quality and craft, and help ship 0-to-1 products — which is directly relevant to a team like this one where design decisions (calendar interaction, booking-conflict feedback, payment-status displays) have to be co-designed with the Go/gRPC domain model, not applied as a CSS layer afterward. Source: [Design at Meta — Product Design](https://www.meta.com/design-at-meta/teams/detail/product-design/), [Design at Meta — Teams](https://www.meta.com/design-at-meta/teams/).

**At Google**, product/UX designers are expected to "work closely with engineers, researchers, and product managers to develop intuitive and innovative solutions that meet user needs," conduct user research themselves rather than outsource it, and communicate design rationale "to leadership, product and design peers, and engineers" — i.e., a senior designer is expected to defend decisions with evidence, not just present mockups. Source: aggregated Google product-design job postings via [Google Careers](https://careers.google.com) (see also [Google Design](https://design.google/) for the org's public design writing and [Material Design](https://m3.material.io/) as the shipped artifact of that practice).

**Career progression** at large design orgs (Google, Meta, Uber, DoorDash, etc.) typically separates an IC ladder (Designer → Senior Designer → Staff/Principal Designer) from a management ladder (Design Manager → Director → VP), with the IC/management fork usually happening around the Senior level. A senior IC designer is expected to work with more autonomy on end-to-end flows, mentor others, and increasingly own systemic/cross-team quality (design-system consistency, accessibility, cross-platform parity) rather than only a single screen. Source: [DoorDash — "Designing" a career ladder for Product Design](https://medium.com/design-doordash/designing-a-career-ladder-for-product-design-8397bd31d3f2) (a public writeup of how one company modeled its ladder explicitly citing Google/Facebook-style precedent); general level structure corroborated by [Harmny — Designer Career Ladder overview](https://harmny.ai/resources/career-ladders/designer) **[secondary aggregator, used only for corroboration, not as a primary claim]**.

**Design-system craft as a first-class deliverable.** Airbnb's Design Language System (DLS), publicly documented by its creator, reframed "design system" away from "a collection of buttons" toward a shared visual *language*: a small cross-functional team of designers and engineers cleared their calendars and worked together in a dedicated space to build it, and the system is now woven directly into the tools teams use daily (e.g., surfaced inside the Figma design panel), not maintained as a separate reference nobody opens. Source: Karri Saarinen, [Building a Visual Language: Behind the Scenes of Our Airbnb Design System](https://medium.com/airbnb-design/building-a-visual-language-behind-the-scenes-of-our-airbnb-design-system-224748775e4e) and [karrisaarinen.com — Creating the Airbnb Design System](https://karrisaarinen.com/posts/building-airbnb-design-system/). Uber runs the same model publicly: **Base** is "the design system that defines the foundations of user interfaces across Uber's ecosystem," implemented as both a documentation site (base.uber.com, built on zeroheight) and a production React library (`baseui`), so design tokens and a shippable component library are the same artifact, not two things kept in sync by hand. Source: [Uber Base design system](https://base.uber.com/), [uber/baseweb on GitHub](https://github.com/uber/baseweb).

**[Synthesis]** For this project specifically: because this is a small team building a vertical slice (per `CLAUDE.md`, Booking is the first bounded context, others "follow the same pattern"), a senior designer's highest-leverage move is the Uber/Airbnb move — define reusable tokens and components *once*, in a form both Figma and code can consume — rather than one-off screens per context (court booking, hosting, joining, community). The polymorphic Booking domain model (recurring-hire / individual / game / competition sharing one aggregate) should have a matching *design* decision: one booking-card / booking-detail component family with variant props, not four bespoke screens.

---

## 2. Core UX competencies

### 2.1 Nielsen Norman Group's 10 usability heuristics

These are the field's most cited heuristic set, originated by Jakob Nielsen and Rolf Molich in 1990 and refined by Nielsen in 1994 after factor-analyzing 249 real usability problems; NN/g maintains them as still current. Source: [NN/g — 10 Usability Heuristics for User Interface Design](https://www.nngroup.com/articles/ten-usability-heuristics/), with a dedicated deep-dive for complex, domain-specific apps (directly relevant to a booking/scheduling system) at [NN/g — 10 Usability Heuristics Applied to Complex Applications](https://www.nngroup.com/articles/usability-heuristics-complex-applications/).

1. **Visibility of system status** — "The system should always keep users informed about what is going on, through appropriate feedback within reasonable time." Source: [NN/g — Visibility of System Status](https://www.nngroup.com/articles/visibility-system-status/). *Applied here:* a booking submission must show pending → confirmed/rejected state, not a silent spinner; a double-booking conflict must surface immediately, matching the backend's `ErrCourtDoubleBooked`.
2. **Match between system and the real world** — use the user's language and real-world conventions, not internal domain jargon. *Applied here:* the domain calls it a `Booking` aggregate with `recurring-hire/individual/game/competition` kinds; end users think "I reserved a court" / "I'm hosting a game" / "I joined a game" — the UI copy should map to *their* mental model even where the underlying model is unified.
3. **User control and freedom** — a clearly marked "emergency exit": undo, cancel, back out of a flow without penalty. *Applied here:* `CancelBooking` exists in the backend (T3) specifically so a booking mistake is recoverable — the UI must make cancellation discoverable and low-friction, not buried.
4. **Consistency and standards** — don't make users wonder if different words, situations, or actions mean the same thing; follow platform and internal conventions.
5. **Error prevention** — "even better than good error messages is careful design which prevents a problem from occurring in the first place" (e.g., disabling/graying already-booked slots in the calendar before the user attempts to select them, rather than only rejecting on submit).
6. **Recognition rather than recall** — surface options and history (e.g., past courts booked, saved payment method) instead of making users remember them.
7. **Flexibility and efficiency of use** — accelerators for expert/repeat users (e.g., a court admin rebooking a recurring slot) without cluttering the interface for novices.
8. **Aesthetic and minimalist design** — every extra unit of information competes with the relevant units and diminishes their visibility.
9. **Help users recognize, diagnose, and recover from errors** — error messages in plain language, precisely indicating the problem, constructively suggesting a solution.
10. **Help and documentation** — even though the system should ideally be usable without it, help content should be easy to search and task-focused.

### 2.2 Information architecture

IA work — how booking, hosting, joining, court browsing, community, and profile content is structured and labeled — should be validated with users' own mental models, not just an engineer's or designer's intuition. **Card sorting** ("participants group individual labels ... in a way that makes sense to them") is NN/g's standard method for generating an IA that matches how users actually categorize content, and **tree testing** is the complementary method for validating navigation *after* it's built. Sources: [NN/g — Card Sorting: Why & When](https://www.nngroup.com/videos/card-sorting-why-when/), [NN/g — Open vs. Closed Card Sorting](https://www.nngroup.com/videos/open-vs-closed-card-sorting/), [NN/g — Card Sorting vs. Tree Testing](https://www.nngroup.com/articles/card-sorting-tree-testing-differences/). **[Synthesis]** For this app, a concrete IA question worth card-sorting: do users think of "my bookings," "games I'm hosting," and "games I've joined" as one list filterable by role, or as three separate destinations? The polymorphic Booking model makes the *unified list* technically natural, but that doesn't mean it's the right user-facing IA — this should be tested, not assumed from the schema.

### 2.3 Interaction design and error handling

NN/g's error-message guidance is specific and actionable, not just "be nice": messages should be *human-readable*, *precise about what happened*, *constructive* (suggest a next step), and *not blame the user*; generic messages like "an error occurred" fail the precision test. Error messages should be displayed *close to the source of the error* (next to the offending field, not only in a toast) so users don't have to hold the problem in working memory while fixing it, and should be visible without being obnoxious (avoid firing before the user has finished input). Sources: [NN/g — Error-Message Guidelines](https://www.nngroup.com/articles/error-message-guidelines/), [NN/g — 10 Design Guidelines for Reporting Errors in Forms](https://www.nngroup.com/articles/errors-forms-design-guidelines/), [NN/g — An Error Messages Scoring Rubric](https://www.nngroup.com/articles/error-messages-scoring-rubric/). **Applied here:** a `CreateBooking` conflict (`23P01`/`ErrCourtDoubleBooked`) reaching the client should never render as a raw gRPC status string — it needs a designed message: "This court is already booked for that time — here are the next available slots," with an actionable recovery path, not just a red banner.

### 2.4 Accessibility — WCAG 2.1/2.2 Level AA

WCAG is organized around four principles — **POUR**: Perceivable, Operable, Understandable, Robust — and conformance levels A/AA/AAA are cumulative (AA conformance requires meeting all A *and* AA criteria). WCAG 2.1 AA is the de facto global compliance bar referenced by most accessibility law and procurement requirements. Source: [W3C — WCAG 2.1](https://www.w3.org/TR/WCAG21/).

Concrete AA criteria most relevant to this product:

- **1.4.3 Contrast (Minimum), Level AA** — normal text needs ≥4.5:1 contrast against its background; large text (≥18pt, or ≥14pt bold) needs ≥3:1. Source: [WebAIM — Contrast and Color Accessibility](https://webaim.org/articles/contrast/); AAA raises this to 7:1/4.5:1 for those who need it.
- **1.4.11 Non-text Contrast, Level AA** — UI components and meaningful graphical objects (icons, form-field borders, calendar-cell boundaries) need ≥3:1 contrast against adjacent colors — this is the criterion that most often catches "elegant" low-contrast icon-only buttons and pale disabled/unavailable calendar cells.
- **2.5.8 Target Size (Minimum), Level AA — new in WCAG 2.2** — interactive targets must be at least 24×24 CSS px, with defined exceptions (adequate spacing to neighboring targets, inline text links, essential presentation). Source: [wcag22aa.org — Target Size (Minimum) SC 2.5.8](https://wcag22aa.org/new-criteria/target-size/). Note this is a *floor*; platform guidance (§3.3 below) is stricter and should govern native apps.
- Keyboard operability, visible focus indicators, and form error identification/labeling round out the AA criteria most load-bearing for a booking flow with calendars, forms, and payment entry.

Since WCAG AA is already a stated non-functional requirement for this project **[per repo context, not independently verified in this research pass]**, accessibility should be a design-review gate, not a post-hoc audit: contrast and target-size checks belong in the same review pass as visual QA, not a separate compliance sweep at the end.

---

## 3. Core UI/visual design competencies

### 3.1 Typography, spacing, and visual hierarchy — *Refactoring UI*

*Refactoring UI* (Adam Wathan & Steve Schoger) is a widely-cited practical reference specifically because it translates abstract design principles into "repeatable, systematic decisions" usable by people without formal design training — relevant here since engineers may be implementing these designs. Key, directly actionable claims from the book:

- **"Most interface problems are hierarchy problems"** — and the fix is almost always to *de-emphasize* secondary elements, not to make the primary element louder.
- **Establish a spacing/sizing *system*** (a constrained scale, e.g. 4/8px steps) rather than ad hoc pixel values — consistent with how Material and Base define spacing tokens (§3.3).
- **Give designs room to breathe** — generous whitespace is one of the fastest fixes for a cluttered screen.
- **Typography discipline**: avoid pure black on white (reduces perceived polish and can be harsher for some vision conditions), limit the type palette to ~2 typefaces, and don't rely on font size alone to convey importance — combine size with weight, color, and spacing.

Source: [Refactoring UI — official site](https://refactoringui.com/); principle summaries corroborated via [GitHub — erikuus/good-ui summary](https://github.com/erikuus/good-ui) and multiple independent book-summary writeups found in research for this dossier.

### 3.2 Component-based design systems — how real systems structure this

**Atomic Design** (Brad Frost, 2013) is the conceptual model underlying most production design systems: **atoms** (buttons, inputs, labels — the smallest functional pieces), **molecules** (atoms grouped into a small functional unit, e.g. a labeled search field), **organisms** (molecules/atoms composed into a distinct section, e.g. a page header or a court-listing card), then **templates** and **pages** as the concrete layouts. It's explicitly a *mental model* for thinking about UI as both a whole and a set of composable parts, not a rigid linear build order. Source: [Brad Frost — Atomic Design, Chapter 2: Methodology](https://atomicdesign.bradfrost.com/chapter-2/).

Real systems apply this with tokens-first thinking:

- **Material Design 3** documents its "Foundations" (layout, color, typography, accessibility, adaptive design across screen sizes) as the base layer beneath components — the system is explicitly meant to be "personal, adaptive, and expressive" while staying systematic. Source: [m3.material.io/foundations](https://m3.material.io/foundations), [m3.material.io](https://m3.material.io/).
- **Uber's Base** is organized the same way — foundational tokens (typography, color, grid, motion) feeding a shared component library (`baseui` on npm) consumed by both design tools and production React code, so there's one source of truth rather than a Figma library and a codebase quietly drifting apart. Source: [base.uber.com](https://base.uber.com/), [uber/baseweb](https://github.com/uber/baseweb).
- **PayPal**, for its internal tooling, adopted Microsoft's **Fluent** design system and used UXPin Merge to let product teams (not just designers) prototype directly against production React components — reportedly enabling ~90% of design projects to ship without dedicated designer involvement on routine work, freeing designers for higher-leverage problems. Source: [UXPin — How PayPal Scaled Their Design Process with UXPin Merge](https://www.uxpin.com/studio/blog/paypal-scaled-design-process-uxpin-merge/). **[Note: PayPal's public-facing design-system branding could not be confirmed by name in this research pass; the Fluent/UXPin claim is the sourced fact, not a specific "PayPal system" name.]**

**[Synthesis]** For this codebase: Vue 3 (web) and SwiftUI (iOS), eventually Kotlin/Compose (Android), means tokens (spacing scale, type scale, color roles, elevation/motion) should be defined *once*, platform-agnostically, and implemented as three parallel component libraries — exactly the Base/Material pattern — rather than three independently "vibe-designed" apps that happen to share a backend. A `BookingCard` design should be one spec with platform-appropriate implementations, not three separate designs.

### 3.3 Platform-native conventions (critical for the Swift/SwiftUI client)

Apple's Human Interface Guidelines are built on three core principles — **Clarity** (legible, precise, uncluttered), **Deference** (UI recedes so content/tasks lead), and **Depth** (layering and motion communicate hierarchy) — plus a fourth practical thread of **Consistency** with platform conventions. Source: [Apple Developer — Human Interface Guidelines](https://developer.apple.com/design/human-interface-guidelines) (corroborated via multiple independent HIG summaries in this research pass, since the site itself blocks automated fetchers).

Concrete, numeric platform guidance a senior designer would enforce:

- **Apple HIG minimum tappable area: 44×44 pt** for any interactive control — Apple's own usability testing found targets smaller than this measurably increase missed taps and frustration. Source: corroborated via [LukeW — Touch Target Sizes](https://www.lukew.com/ff/entry.asp?1085=) and multiple HIG summaries; this is stricter than WCAG's 24×24 CSS px floor and should be the actual bar for the iOS app.
- **Material Design: minimum 48×48 dp** touch targets, translating to roughly the same ~9mm physical size regardless of screen density, with ≥8dp spacing between adjacent targets. Source: [Android Developers — Touch target size](https://support.google.com/accessibility/android/answer/7101858), [Material Design 1 — Accessibility/Usability](https://m1.material.io/usability/accessibility.html).
- Both platform minimums exceed the WCAG 2.2 AA floor of 24×24 CSS px — **[Synthesis]** meaning "meets WCAG AA" is a necessary but not sufficient bar for native mobile; the platform HIG/Material numbers should be the working standard for iOS/Android, with WCAG AA as the non-negotiable minimum enforced everywhere including web.

---

## 4. Domain-specific application — booking/marketplace + community app

**[Synthesis section — applying the sourced principles above to this product's actual surfaces.]**

### 4.1 Booking flows and availability calendars

- **Error prevention over error correction (NN/g heuristic #5):** the calendar itself should visually distinguish available / booked / user's-own-booking / (if applicable) blocked-by-admin slots *before* the user attempts a selection, the same pattern Airbnb uses — greyed-out, non-interactive cells for unavailable dates rather than allowing a selection that then fails on submit. Source pattern: [Airbnb Community — what greyed-out calendar dates mean](https://community.withairbnb.com/t5/Community-cafe/Booking-What-does-the-greyed-out-dates-mean/td-p/1588703) (illustrates the real-world convention users already expect from a booking calendar).
- Because the backend enforces the no-double-booking invariant with a Postgres `EXCLUDE` constraint as the *authoritative* source of truth (per `CLAUDE.md`), the UI must treat a booking-conflict rejection as an expected, designed-for outcome (race condition between two users booking the same slot), not an exceptional crash state — this is a direct application of heuristic #9 (help users recover from errors): show the conflict, then immediately re-query and re-render nearby available alternatives.
- **Recurring bookings** need a distinct interaction pattern from one-off bookings even though they share one domain aggregate — real calendar/booking UIs (and NN/g's complex-application heuristics guidance) support this by making recurrence an explicit, visible modifier on the booking flow rather than inferring it, and by giving users a clear affordance to edit/cancel "this occurrence" vs. "the whole series."

### 4.2 Payment / paid-unpaid status

- Status should be a **scannable, single-purpose signal** — badge UI conventions favor one content type per badge (a label, a count, or an icon, not a mix) and reserve solid/filled treatment for states requiring attention (e.g., "Unpaid," "Payment failed") versus a quieter, lower-contrast treatment for resolved/passive states (e.g., "Paid"). **[Synthesis, informed by general badge-pattern conventions]** — verify against whatever component library is chosen, but the underlying rule (urgency = higher visual weight) is a direct application of heuristic #8 (aesthetic and minimalist design: don't let "Paid" compete visually with "Unpaid — action needed").
- Since this project's payments model is explicitly Stripe *and* offline entry as "one source of truth" (`CLAUDE.md`), the UI must never let the two payment paths look like two different truths — a booking paid offline by a Game Admin must render identically (same "Paid" state, same receipt/record affordance) as one paid via Stripe, or the UI itself introduces a trust bug even if the backend is correct. This is heuristic #4, consistency and standards, applied to a business-logic seam rather than just button styles.
- Contrast for status color-coding (green=paid, red=unpaid, etc.) must not rely on hue alone — WCAG 1.4.1 (Use of Color, Level A) and general accessibility practice require a redundant cue (icon, label text, pattern) so colorblind users aren't dependent on red/green discrimination.

### 4.3 Roster / game views

- Borrowing directly from sports-team-management UX conventions (TeamSnap-style RSVP/roster patterns): the core jobs are *availability polling* (who's in/out), *roster visibility* (who's confirmed), and *fast recognition over recall* (heuristic #6) — a coach or Game Admin should see committed players at a glance without opening each profile. Source: general pattern description via [sports-league-software UX summaries](https://zipdo.co/best/sports-team-scheduling-software/) **[secondary/aggregated source, used for pattern corroboration only]**.
- Given this platform's matchmaking design decision ("automated from history, always manually overridable" per `CLAUDE.md`), the roster/game view needs a clear, low-friction override affordance — automation should visibly suggest, not silently decide, echoing heuristic #3 (user control and freedom) and heuristic #1 (visibility of system status: the user should be able to see *why* the system suggested a given matchup, not just the result).

### 4.4 Mobile-first / native iOS considerations

- SwiftUI implementation should default to platform-native components (native date/time pickers, native navigation patterns, native alerts) rather than reimplementing web patterns — Apple's Deference principle exists precisely so users get an interface that behaves the way the rest of iOS already taught them to expect, lowering learning cost for a scheduling-heavy app used in short, often one-handed sessions courtside.
- Given real usage context (checking court availability outdoors, possibly one-handed, sometimes with sun glare or gloves in cooler climates), targets should lean toward the *generous* end of the 44pt Apple minimum, and critical actions (confirm booking, cancel) should not require precision taps near screen edges.
- Cross-platform parity is a design-system problem, not a per-screen problem (§3.2): the safest way to guarantee Vue/SwiftUI/Kotlin apps don't visually and behaviorally diverge is one shared token/spec source feeding three implementations, per the Base/Material model.

---

## 5. Decision heuristics for design review — concrete, actionable checks

Use these as a review checklist; each is either sourced or marked synthesis.

| Check | Standard | Source |
|---|---|---|
| Interactive targets ≥44×44pt (iOS), ≥48×48dp (Android), ≥24×24 CSS px minimum everywhere (WCAG floor) | Apple HIG / Material / WCAG 2.5.8 | [HIG](https://developer.apple.com/design/human-interface-guidelines), [Android a11y](https://support.google.com/accessibility/android/answer/7101858), [wcag22aa.org](https://wcag22aa.org/new-criteria/target-size/) |
| Normal text contrast ≥4.5:1, large text/UI components ≥3:1 | WCAG 1.4.3 / 1.4.11 AA | [WebAIM](https://webaim.org/articles/contrast/) |
| Every async action shows immediate feedback (spinner/skeleton/optimistic state) within a reasonable time | NN/g heuristic #1 | [NN/g — Visibility of System Status](https://www.nngroup.com/articles/visibility-system-status/) |
| Full-page loads use skeleton screens (2–10s); isolated components use spinners; long operations (>10s) use progress bars with real progress | NN/g loading-pattern guidance | [NN/g — Skeleton Screens 101](https://www.nngroup.com/articles/skeleton-screens/), [NN/g — Skeleton vs Progress vs Spinner](https://www.nngroup.com/videos/skeleton-screens-vs-progress-bars-vs-spinners/) |
| Error messages sit next to the field/action that caused them, state the problem in plain language, and suggest a next step — never a bare "An error occurred" | NN/g error-message guidelines | [NN/g — Error-Message Guidelines](https://www.nngroup.com/articles/error-message-guidelines/), [NN/g — Errors in Forms](https://www.nngroup.com/articles/errors-forms-design-guidelines/) |
| Destructive/high-consequence actions (cancel booking, unrecoverable state changes) have an undo or confirmation path | NN/g heuristic #3 | [NN/g — 10 Usability Heuristics](https://www.nngroup.com/articles/ten-usability-heuristics/) |
| Empty states (no bookings yet, no games nearby) explain *what belongs here* and give a direct path to the first action — never a bare blank screen | NN/g empty-state guidance | [NN/g — Empty States in Application Design](https://www.nngroup.com/videos/empty-states-in-application-design-guidelines/) |
| Color is never the sole carrier of meaning (paid/unpaid, available/booked) — pair with icon/label/pattern | WCAG 1.4.1 Use of Color (A) | [W3C WCAG 2.1](https://www.w3.org/TR/WCAG21/) |
| All interactive elements reachable and operable by keyboard, with a visible focus indicator (web) | WCAG 2.1.1 / 2.4.7 AA | [W3C WCAG 2.1](https://www.w3.org/TR/WCAG21/) |
| Spacing follows a defined scale (e.g. 4/8px steps), not arbitrary one-off values | Refactoring UI / token-based systems | [Refactoring UI](https://refactoringui.com/), [Material foundations](https://m3.material.io/foundations) |
| A single component spec (tokens + variants) backs each cross-platform pattern (e.g. BookingCard), rather than independently designed per-platform screens | Base/Material/Atomic Design pattern | [Uber Base](https://base.uber.com/), [Brad Frost — Atomic Design](https://atomicdesign.bradfrost.com/chapter-2/) — **[synthesis applying the pattern to this repo]** |
| Calendar/availability UI prevents invalid selections visually rather than only rejecting on submit | NN/g heuristic #5 (error prevention) | [NN/g — 10 Usability Heuristics](https://www.nngroup.com/articles/ten-usability-heuristics/) |

---

## 6. Anti-patterns to push back on

- **Deceptive patterns ("dark patterns").** Harry Brignull, who coined the term in 2010, catalogs recurring manipulative patterns: *forced continuity* (silently converting a free trial to paid), *confirmshaming* (guilt-tripping copy on decline buttons), *roach motel* (easy to get into a commitment, hard to get out — e.g. easy signup, buried cancellation), *hidden costs* (fees revealed only at the final step), *misdirection*, and *trick questions* in forms. These aren't hypothetical for a booking/payments product — a "cancel booking" flow that's deliberately harder to find than "book" is a roach motel; a Stripe checkout that reveals a service fee only at the last screen is hidden costs. Brignull's taxonomy has since been referenced directly in the EU's Digital Services Act/Digital Markets Act and California's CPRA, meaning these aren't just bad UX, they're increasingly *regulated*. Source: [deceptive.design](https://www.deceptive.design/), [Dr. Harry Brignull bio](https://deceptive.design/about-us/dr-harry-brignull/), also covered by NN/g: [Deceptive Patterns in UX: How to Recognize and Avoid Them](https://www.nngroup.com/articles/deceptive-patterns/).
- **Inconsistent components across platforms.** Shipping a Vue web calendar and a SwiftUI calendar that look and behave differently for the same domain concept (an available court slot) violates heuristic #4 (consistency and standards) and defeats the entire point of a shared design-token system (§3.2/§3.3) — this is the single most likely failure mode for a three-client, one-backend product like this one if design isn't specified once and enforced across implementations. **[Synthesis]**
- **Accessibility as an afterthought / audit-only accessibility.** Treating WCAG AA as a checklist run once before ship, rather than a design-review gate applied continuously, reliably produces regressions (a later-added status badge with 2.8:1 contrast, a new icon-only button under 44pt). Given this project already lists WCAG AA as a stated NFR, retrofitting is strictly more expensive than designing to the bar from the first mockup. **[Synthesis, informed by §2.4/§5]**
- **Letting the domain model dictate the IA uncritically.** The backend's polymorphic Booking aggregate is the *right* engineering decision (per `CLAUDE.md`'s locked decisions) but should not be copy-pasted into the user-facing information architecture without validation — "one Booking type" is a data-model convenience, not evidence that users think of hosting, joining, and reserving as the same mental category (§2.2). A senior designer pushes back on any 1:1 schema-to-screen mapping that hasn't been checked against real user mental models.
- **Silent or unexplained system decisions**, especially in matchmaking ("automated from history, always manually overridable" per `CLAUDE.md`) — an algorithm that assigns a skill level or matchup without showing its reasoning or an easy override violates heuristic #1 (visibility of system status) and heuristic #3 (user control and freedom), and erodes trust in exactly the kind of community feature where trust is the product. **[Synthesis]**
- **Generic error surfaces.** Rendering raw backend/gRPC error text (e.g., a bare `23P01` or `ErrCourtDoubleBooked` string) to end users fails NN/g's error-message guidelines outright — precise-to-the-user, not precise-to-the-database. **[Synthesis, direct application of §2.3]**

---

## 7. Sources

**Nielsen Norman Group (heuristics, IA, errors, loading states, empty states, deceptive patterns)**
- [10 Usability Heuristics for User Interface Design](https://www.nngroup.com/articles/ten-usability-heuristics/)
- [10 Usability Heuristics Applied to Complex Applications](https://www.nngroup.com/articles/usability-heuristics-complex-applications/)
- [Visibility of System Status (Usability Heuristic #1)](https://www.nngroup.com/articles/visibility-system-status/)
- [Error-Message Guidelines](https://www.nngroup.com/articles/error-message-guidelines/)
- [10 Design Guidelines for Reporting Errors in Forms](https://www.nngroup.com/articles/errors-forms-design-guidelines/)
- [An Error Messages Scoring Rubric](https://www.nngroup.com/articles/error-messages-scoring-rubric/)
- [Skeleton Screens 101](https://www.nngroup.com/articles/skeleton-screens/)
- [Skeleton Screens vs. Progress Bars vs. Spinners (Video)](https://www.nngroup.com/videos/skeleton-screens-vs-progress-bars-vs-spinners/)
- [Empty States in Application Design: 3 Guidelines (Video)](https://www.nngroup.com/videos/empty-states-in-application-design-guidelines/)
- [Deceptive Patterns in UX: How to Recognize and Avoid Them](https://www.nngroup.com/articles/deceptive-patterns/)
- [Card Sorting: Why & When (Video)](https://www.nngroup.com/videos/card-sorting-why-when/)
- [Open vs. Closed Card Sorting (Video)](https://www.nngroup.com/videos/open-vs-closed-card-sorting/)
- [Card Sorting vs. Tree Testing](https://www.nngroup.com/articles/card-sorting-tree-testing-differences/)

**Accessibility standards**
- [W3C — Web Content Accessibility Guidelines (WCAG) 2.1](https://www.w3.org/TR/WCAG21/)
- [wcag22aa.org — Target Size (Minimum), SC 2.5.8 (AA)](https://wcag22aa.org/new-criteria/target-size/)
- [WebAIM — Contrast and Color Accessibility](https://webaim.org/articles/contrast/)
- [Android Developers / Google — Touch target size](https://support.google.com/accessibility/android/answer/7101858)

**Company design systems and design orgs**
- [Design at Meta — Product Design](https://www.meta.com/design-at-meta/teams/detail/product-design/)
- [Design at Meta — Teams overview](https://www.meta.com/design-at-meta/teams/)
- [Material Design 3 — Foundations](https://m3.material.io/foundations)
- [Material Design 3](https://m3.material.io/)
- [Material Design 1 — Accessibility/Usability](https://m1.material.io/usability/accessibility.html)
- [Uber Base design system](https://base.uber.com/)
- [Uber Base — Content design](https://base.uber.com/6d2425e9f/p/4245c4-content-design)
- [uber/baseweb (React implementation of Base) — GitHub](https://github.com/uber/baseweb)
- [UXPin — How PayPal Scaled Their Design Process with UXPin Merge (Fluent design system)](https://www.uxpin.com/studio/blog/paypal-scaled-design-process-uxpin-merge/)
- Karri Saarinen, [Building a Visual Language: Behind the Scenes of Our Airbnb Design System](https://medium.com/airbnb-design/building-a-visual-language-behind-the-scenes-of-our-airbnb-design-system-224748775e4e)
- [karrisaarinen.com — Creating the Airbnb Design System](https://karrisaarinen.com/posts/building-airbnb-design-system/)
- [Google Careers](https://careers.google.com) / [Google Design](https://design.google/) (aggregated product-design role descriptions)
- [DoorDash — "Designing" a career ladder for Product Design](https://medium.com/design-doordash/designing-a-career-ladder-for-product-design-8397bd31d3f2)

**Platform guidelines**
- [Apple Developer — Human Interface Guidelines](https://developer.apple.com/design/human-interface-guidelines)
- [LukeW — Touch Target Sizes](https://www.lukew.com/ff/entry.asp?1085=)

**Visual/UI craft references**
- [Refactoring UI (Adam Wathan & Steve Schoger) — official site](https://refactoringui.com/)
- [Brad Frost — Atomic Design, Chapter 2: Methodology](https://atomicdesign.bradfrost.com/chapter-2/)

**Deceptive/dark patterns**
- [deceptive.design (formerly darkpatterns.org)](https://www.deceptive.design/)
- [Dr. Harry Brignull — bio](https://deceptive.design/about-us/dr-harry-brignull/)

**Domain-pattern corroboration (secondary sources, used only to corroborate real-world conventions, not as primary claims)**
- [Airbnb Community — what greyed-out calendar dates mean](https://community.withairbnb.com/t5/Community-cafe/Booking-What-does-the-greyed-out-dates-mean/td-p/1588703)
- [ZipDo — sports team scheduling software UX patterns](https://zipdo.co/best/sports-team-scheduling-software/)
- [Harmny — Designer career-ladder overview](https://harmny.ai/resources/career-ladders/designer)

---

*Compiled for use as loaded context when briefing an AI subagent in a Senior UX/UI Designer persona reviewing this repository's Vue/SwiftUI/Kotlin clients. Sections marked [Synthesis] are this document's own reasoning applying the sourced material above to this specific product and should be treated as informed direction, not as independently verified external fact.*
