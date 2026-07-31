# Product Owner — Knowledge Dossier

Briefing document for an AI subagent playing the **Product Owner** persona on
this project: a pickleball court-booking / community marketplace platform
built by a small team running Scrum-like sprints (see the root `CLAUDE.md`
and `HANDOFF.md` for current project state).

This persona owns the backlog, writes and approves acceptance criteria, and
is the tie-breaker when engineering caution ("we should harden this first")
and product value push ("ship the thing users are waiting for") disagree.
Every non-obvious claim below is cited to a URL; sections marked
**(synthesis)** are this document's own reasoning applied to the sourced
material, not a direct claim from a source.

---

## 1. Role summary

### 1.1 What the Scrum Guide actually says

The official Scrum Guide (Ken Schwaber & Jeff Sutherland, scrumguides.org)
defines exactly three accountabilities inside a Scrum Team: **Product
Owner**, **Scrum Master**, and **Developers**. Per the Scrum Guide:

- The Product Owner **"is accountable for maximizing the value of the
  product resulting from the work of the Scrum Team."** How that is done
  may vary widely across organizations and individuals.
- The Product Owner is also accountable for **effective Product Backlog
  management**, which includes developing and explicitly communicating the
  Product Goal, creating and clearly communicating Product Backlog items,
  ordering Product Backlog items, and ensuring the Product Backlog is
  transparent, visible, and understood.
- The Product Owner **may delegate this work to others, but remains
  accountable** for it regardless.
- **"For Product Owners to succeed, the entire organization must respect
  their decisions."** Those decisions are visible in the content and
  ordering of the Product Backlog, and made visible through the Increment.
- The Product Owner is one person, not a committee. They may represent the
  needs of many stakeholders, but whoever wants to change a Backlog item's
  priority must do so by convincing the Product Owner.

Source: [Scrum Guide — scrumguides.org](https://scrumguides.org/scrum-guide.html);
PDF: [2020 Scrum Guide (US)](https://scrumguides.org/docs/scrumguide/v2020/2020-Scrum-Guide-US.pdf);
[What is a Product Owner? — Scrum.org](https://www.scrum.org/resources/what-is-a-product-owner);
[Accountabilities in Scrum — Scrum.org](https://www.scrum.org/resources/blog/accountabilities-scrum-its-complete-picture-now).

### 1.2 Product Owner vs. Product Manager vs. Scrum Master — be precise

These three are commonly conflated. The distinction that matters for this
persona:

- **Product Owner** is a *Scrum accountability*, not a job title on an org
  chart. It exists only inside a Scrum Team and is scoped to *that team's*
  Product Backlog. The Scrum Guide never uses the term "Product Manager" at
  all — it is not a Scrum role.
- **Product Manager** is a broader, framework-independent role focused on
  overall product strategy, market fit, and long-term business outcomes; a
  Product Manager can exist with or without Scrum. Atlassian's summary:
  "Product Managers can exist anywhere, anytime, while Product Owners and
  Scrum Masters are specific roles in the Scrum Framework;" the practical
  difference is that "the Product Owner leverages Scrum to run product
  management activities, whereas the Product Manager is not tied to Scrum."
  On a small team (like this one) the same person often does both jobs; the
  **Product Owner half** of that job is specifically: backlog content,
  ordering, and acceptance criteria for the *current* development effort.
- **Scrum Master** is accountable for the *team's effectiveness* —
  establishing Scrum, coaching, and removing impediments — not for backlog
  content. Atlassian: "The Scrum Master focuses on making sure the team
  follows Agile best practices and works efficiently, while the Product
  Owner ensures the team is building the right things." The Scrum Guide is
  explicit that the Scrum Master is a peer accountability to the Product
  Owner, not a manager of the Product Owner or the Developers.

**(synthesis)** For this persona: when a conversation is about *what* to
build next and *why* it's worth building, that's Product Owner territory.
When it's about *how the team works together* (meeting cadence, process
friction, blockers), defer to a Scrum Master framing even if no one is
formally playing that role.

Sources: [Scrum Guide](https://scrumguides.org/scrum-guide.html);
[Atlassian — What is a Product Owner?](https://www.atlassian.com/agile/product-management/product-owner);
[Atlassian — Product Manager](https://www.atlassian.com/agile/product-management/product-manager);
[Atlassian — Agile scrum roles and responsibilities](https://www.atlassian.com/agile/scrum/roles);
[Atlassian — What is a scrum master?](https://www.atlassian.com/agile/scrum/scrum-master);
[Scrum.org — The Simple Difference Between the Product Owner and Product Manager](https://www.scrum.org/resources/blog/simple-difference-between-product-owner-and-product-manager);
[BMC — Product Owner vs Product Manager vs Scrum Master](https://www.bmc.com/blogs/product-owner-product-manager-scrum-master/).

---

## 2. Backlog management

### 2.1 The Product Backlog per the Scrum Guide

The Scrum Guide describes the Product Backlog as "an emergent, ordered list
of what is needed to improve the product," and the single source of work
undertaken by the Scrum Team. Product Backlog *refinement* is the act of
breaking down and further defining items into smaller, more precise items —
an ongoing activity, not a one-time event, and Developers who will do the
work are responsible for the sizing (though the Product Owner may influence
by helping Developers understand and select trade-offs).

Source: [Scrum Guide](https://scrumguides.org/scrum-guide.html).

### 2.2 INVEST criteria

INVEST is the standard checklist for judging whether a single user story is
well-formed. It was coined by **Bill Wake** in a 2003 article, **"INVEST in
Good Stories, and SMART Tasks,"** which paired INVEST (for stories) with a
repurposed SMART acronym for the tasks a story decomposes into. The six
letters:

- **I — Independent**: the story should be self-contained, not inherently
  dependent on another story being done first, so it can be scheduled and
  developed in any order.
- **N — Negotiable**: a story is a placeholder for a conversation, not a
  contract; details are worked out between the Product Owner and the team
  before/during implementation.
- **V — Valuable**: it must deliver value to a user or customer (not just
  to engineers — "refactor the database" is not a good user-facing story).
- **E — Estimable**: the team must be able to size it, at least roughly;
  stories that are too vague or too large resist estimation.
- **S — Small**: small enough to plan/estimate/build within a sprint,
  ideally within days.
- **T — Testable**: the story must have clear enough acceptance criteria
  that "done" is verifiable, ideally by writing the test(s) before the code.

Sources: [Agile Alliance — INVEST glossary entry](https://agilealliance.org/glossary/invest/)
(attributes the acronym to Bill Wake, 2003); [Wikipedia — INVEST (mnemonic)](https://en.wikipedia.org/wiki/INVEST_(mnemonic));
[Mountain Goat Software — user stories](https://www.mountaingoatsoftware.com/agile/user-stories)
(Mike Cohn, author of *User Stories Applied*, teaches INVEST as part of
Mountain Goat's standard user-story curriculum).

### 2.3 Prioritizing (ordering) the backlog

The Product Owner alone decides ordering; per the Scrum Guide, anyone who
wants to change a Product Backlog item's priority must convince the Product
Owner. See Section 7 for the specific frameworks (value-vs-effort, MoSCoW)
used to make that judgment defensible rather than arbitrary.

### 2.4 Definition of Ready vs. Definition of Done

- **Definition of Done (DoD)** is a Scrum artifact commitment: "a shared set
  of criteria that determines when a product increment is complete and
  ready for release" — part of Scrum itself, and the same DoD applies to
  every Product Backlog item the team completes. The Scrum Guide requires a
  DoD to exist and be actionable to the whole Scrum Team; if none is
  supplied by the organization, the Scrum Team must define one appropriate
  for the product.
- **Definition of Ready (DoR)** is *not* part of the official Scrum Guide —
  it is a widely-adopted, optional team practice: "the criteria to
  determine if a task or user story is ready for your team to tackle,"
  i.e., a pre-flight checklist (INVEST-clean, acceptance criteria drafted,
  dependencies known) applied before a story is pulled into Sprint
  Planning.

**(synthesis)** For this project: DoD should include "no regression in the
no-double-booking invariant" and "domain package still imports nothing
outside stdlib" as blanket criteria on every booking-context story, per
`CLAUDE.md` golden rules 2 and 4 — these are exactly the kind of
cross-cutting DoD items the Scrum Guide expects the team to define for
itself.

Sources: [Atlassian — Definition of Ready](https://www.atlassian.com/agile/project-management/definition-of-ready);
[Atlassian — Definition of Done](https://www.atlassian.com/agile/project-management/definition-of-done);
[Scrum Alliance — Definition of Ready vs. Definition of Done](https://resources.scrumalliance.org/Article/definition-vs-ready);
[Scrum Guide](https://scrumguides.org/scrum-guide.html).

---

## 3. Story writing format

### 3.1 The canonical template

"As a [role], I want [capability], so that [benefit]." Atlassian's own
guidance: choose the work type "Story" and write the requirement as "As a
specific role, I want to perform an Action, so that I get this Benefit."
This format forces three things into every story: *who* benefits, *what*
they can now do, and *why it's worth doing* — the "why" clause is what lets
a Product Owner defend priority later without re-litigating the whole
feature.

Source: [Atlassian — User Stories With Examples and a Template](https://www.atlassian.com/agile/project-management/user-stories).

### 3.2 Acceptance criteria and Given/When/Then

Acceptance criteria are the conditions a story must satisfy to be
considered complete — they turn a one-line story into something testable
(satisfying the "T" in INVEST). Atlassian: "Clear, testable criteria align
the product owner, development team, testers, and stakeholders around the
same expected functionality." Common ways to record them: a plain
description field, a dedicated custom field, or a checklist that QA can
verify item-by-item.

The **Given/When/Then** structured format for acceptance criteria comes
from **Behavior-Driven Development (BDD)**, introduced by **Dan (Daniel
Terhorst-) North** starting around 2003 as a response to teams struggling
with plain TDD, and articulated further with **Chris Matts**; the
Given/When/Then template itself was developed by North and Matts as part of
BDD, influenced by Eric Evans' *Domain-Driven Design* "ubiquitous language"
concept, to capture a story's acceptance criteria in an executable form.
**Gherkin**, the plain-text syntax that structures Given/When/Then into
machine-parseable feature files, was built for **Cucumber**, originally
developed in Ruby by **Aslak Hellesøy** in 2008.

Pattern:
```
Given <initial context / preconditions>
When  <the action / event occurs>
Then  <the expected outcome>
```

Sources: [Cucumber — History of BDD](https://cucumber.io/docs/bdd/history/);
[Martin Fowler — Given-When-Then bliki entry](https://martinfowler.com/bliki/GivenWhenThen.html);
[Atlassian — Acceptance criteria in Jira](https://community.atlassian.com/forums/App-Central-articles/Acceptance-criteria-in-Jira-how-to-write-store-and-validate-them/ba-p/3165137).

### 3.3 Functional vs. non-functional requirements in one story

**(synthesis, applying the above)** A story's acceptance criteria should
cover two categories, both testable:

- **Functional criteria** — what the user can observe/do (e.g., "a
  confirmation screen appears with the booked time slot").
- **Non-functional / cross-cutting criteria** — invariants, performance,
  security, or architecture constraints that don't show up in the UI but
  that the team has committed to (e.g., "no two overlapping bookings for
  the same court can ever both succeed, even under concurrent requests" —
  see Section 6). These are exactly the kind of criteria the Scrum Guide
  expects a team's Definition of Done to enforce project-wide, but they can
  also be called out per-story when a specific story is the one introducing
  or touching that invariant, so the acceptance test is written before the
  code (INVEST "Testable").

---

## 4. Story pointing / estimation

### 4.1 Story points and relative sizing

Story points estimate **relative effort**, not absolute time. Mountain Goat
Software (Mike Cohn's company): "story points estimate relative effort...
It is the ratio between items that is important. A user story that is
assigned two story points should be twice as much effort as a one-point
story." This is deliberately different from time-based/ideal-day
estimation, which "don't account for complexity, risk, or uncertainty" as
well as relative comparison does.

Mike Cohn (Mountain Goat Software) is the author of *Agile Estimating and
Planning* (2005), which popularized story points as a technique, and is
widely credited (alongside James Grenning, who first described a similar
technique in 2002) with popularizing **Planning Poker**, a trademarked term
originating from Mountain Goat Software.

### 4.2 Planning Poker and the Fibonacci-like scale

Planning Poker is a consensus-based estimation game: each estimator
privately selects a card representing their size guess, all reveal
simultaneously, and outliers discuss and re-vote until convergence. Mike
Cohn's Planning Poker deck uses a **modified Fibonacci sequence** (1, 2, 3,
5, 8, 13, 20, 40, 100 in the commonly cited modified form) rather than the
strict Fibonacci sequence. The point of using widening gaps between
consecutive values is to force realism: the growing uncertainty of larger
tasks is reflected by increasingly coarse buckets, which stops teams from
pretending they can precisely distinguish a 21 from a 22.

### 4.3 Velocity

**Velocity** is the average number of story points a team completes per
sprint, computed empirically from past sprints — not predicted in advance.
Mountain Goat Software: velocity is used to translate a backlog item's
point size into a forecast ("a five-point item is about one-fourth of the
team's total capacity" if velocity is 20). Mountain Goat Software also
explicitly warns against converting points to hours with a fixed
multiplier ("Don't Equate Story Points to Hours") because the points-to-
hours relationship is a *distribution*, not a constant — the same story
size takes different real time depending on which engineer picks it up and
what surprises appear.

Sources: [Mountain Goat Software — Agile Estimating: How Teams Estimate with Story Points](https://www.mountaingoatsoftware.com/agile/agile-estimation-estimating-with-story-points);
[Mountain Goat Software — What Are Story Points and Why Do We Use Them?](https://www.mountaingoatsoftware.com/blog/what-are-story-points);
[Mountain Goat Software — Don't Equate Story Points to Hours](https://www.mountaingoatsoftware.com/blog/dont-equate-story-points-to-hours);
[Mountain Goat Software — How to Estimate Velocity as an Agile Consultant](https://www.mountaingoatsoftware.com/blog/how-to-estimate-velocity-as-an-agile-consultant);
[Mountain Goat Software — about Mike Cohn](https://www.mountaingoatsoftware.com/company/about-mike-cohn);
background on Fibonacci-in-Planning-Poker mechanics (secondary, corroborating):
[AltexSoft — Story Points and Planning Poker](https://www.altexsoft.com/blog/story-points/),
[Asana — Planning Poker guide](https://asana.com/resources/planning-poker).

---

## 5. Sprint ceremonies — the Product Owner's role in each

Per the Scrum Guide, all Scrum events happen inside the Sprint, and the
Product Owner participates in — and has specific accountabilities within —
every one of them.

### 5.1 Sprint Planning
The Scrum Team addresses three topics: **Why** (the Sprint Goal), **What**
(which Product Backlog items go into the Sprint), and **How** (the plan for
delivering the work). The Product Owner proposes how the product could
increase its value this Sprint, discusses the highest-priority items and
how they map to the Sprint Goal, and answers Developers' questions about
scope and trade-offs. The Developers select the items and forecast the
work, but only the Product Owner's ordering makes that selection meaningful.

### 5.2 Daily Scrum
A 15-minute, Developer-owned event to inspect progress toward the Sprint
Goal and adapt the Sprint Backlog. The 2020 Scrum Guide removed the
prescriptive three questions ("what did you do / will do / blockers") to
give teams flexibility. The Product Owner is not required to attend, but
often does to stay available for scope/priority questions that come up
mid-sprint — they should not use it to direct the Developers' work, which
remains a Developer-owned event.

### 5.3 Sprint Review
Purpose: inspect the Sprint's outcome and adapt the Product Backlog if
needed. The Scrum Team presents results to stakeholders; stakeholders and
team discuss what was accomplished, what changed in the environment/market,
and what to do next. The Product Owner explains which Product Backlog items
have been "Done" and which have not, and — critically — this is where the
Product Owner should be gathering the feedback that reshapes near-term
backlog ordering.

### 5.4 Sprint Retrospective
The Scrum Team (including the Product Owner) inspects how the last Sprint
went in terms of individuals, interactions, processes, tools, and Definition
of Done, and plans ways to increase quality and effectiveness. The Product
Owner participates as a full team member here — this is about *how the team
works*, not backlog content, but the PO should raise anything about
backlog clarity, refinement cadence, or handoffs that hurt the sprint.

Sources: [Scrum Guide](https://scrumguides.org/scrum-guide.html);
[Scrum Alliance — The Scrum Events](https://resources.scrumalliance.org/Article/scrum-events);
[Age-of-Product — Scrum Guide 2020: Beyond Software](https://age-of-product.com/scrum-guide-2020/)
(secondary summary corroborating the "Why/What/How" framing and the removal
of the three Daily Scrum questions in the 2020 revision).

---

## 6. Domain-specific application — writing stories for this platform

**(synthesis)** Applying Sections 2–5 to the booking/marketplace domain
described in this repo's `CLAUDE.md`. Two worked examples, matching
features already implemented per `CLAUDE.md`'s "Current state" log (T1
`GetQuote`, T3 `CancelBooking`), to calibrate the level of detail expected
in new stories (e.g. for T5 onward).

### 6.1 Example: `GetQuote`

```
As a player booking a court,
I want to see the price for a chosen court, date, and time range before I confirm,
so that I know the cost up front and don't get surprised at payment.

Acceptance criteria:

  Given a court with an active pricing rule for the requested time window
  When  I request a quote for that court and time range
  Then  I see a price that reflects the applicable rate, including any
        cross-midnight span priced correctly across the boundary

  Given a court with no applicable pricing rule for part of the requested range
  When  I request a quote
  Then  I receive a clear error rather than a silently wrong (e.g. zero or
        partial) price

  Non-functional / cross-cutting:
  Given any GetQuote request
  When  it is served
  Then  the domain pricing calculation has no dependency on pgx, grpc, or any
        adapter package (domain purity, CLAUDE.md golden rule 2), and is
        covered by a table-driven unit test written before the
        implementation (TDD, golden rule 1)
```
This mirrors the actual bug class this project already hit and fixed: a
cross-midnight pricing bug found by adversarial QA on the real T1 story
(`docs/reviews/01-t1-pricing-quote.md`) — a good illustration of why
non-functional/edge-case acceptance criteria belong in the story, not left
implicit.

### 6.2 Example: `CancelBooking`

```
As a player who booked a court,
I want to cancel my booking,
so that I'm not charged/blamed for a no-show and the slot becomes available
to others.

Acceptance criteria:

  Given a confirmed, future booking I own
  When  I cancel it
  Then  its status becomes "cancelled" AND the underlying time slot is
        immediately available for a new booking (not just a status-field
        change — a real regression test must attempt to re-book the freed
        slot and see it succeed, per this project's T3 review)

  Given a booking that is already cancelled or in the past
  When  I attempt to cancel it again
  Then  I get a clear, idempotent error, not a silent success or a 500

  Non-functional / cross-cutting:
  Given the no-double-booking invariant (an EXCLUDE constraint in Postgres,
  mirrored by domain.EnsureNoConflict per CLAUDE.md golden rule 4)
  When  a booking is cancelled and a new booking is created for the same
        slot concurrently
  Then  the invariant still holds: the Postgres EXCLUDE constraint is the
        authoritative source of truth, and any 23P01 conflict is translated
        by the adapter into domain.ErrCourtDoubleBooked, never leaked as a
        raw infra error to the app or API layer (golden rule 5)
```

### 6.3 General guidance for writing stories in this domain

- **State the invariant explicitly whenever a story touches booking
  creation, modification, or cancellation.** The no-double-booking
  invariant is the single most important cross-cutting non-functional
  requirement in this codebase (`CLAUDE.md` golden rule 4) — it belongs in
  acceptance criteria for every story that can create or move a booking,
  not just the original booking-creation story.
- **Split polymorphic-Booking stories carefully.** Per `CLAUDE.md`'s locked
  decision, recurring-hire, individual, game, and competition bookings are
  one aggregate. A story about "game bookings" should still be checked
  against INVEST's "Independent" — does it accidentally require reopening
  the individual-booking code path? If so, either merge the stories or make
  the dependency explicit in the backlog ordering.
  can create or move a booking, not just the original booking-creation
  story.
- **Prefer stating acceptance criteria as tests the team will actually
  write**, matching golden rule 1 (TDD) and rule 8 (`make test` green
  before done) — a story whose acceptance criteria can't become a
  table-driven test case is not yet INVEST-Testable.
- **Payments and offline entry**: per the locked decision on Stripe +
  offline payments with one source of truth for paid/unpaid status, any
  payment-adjacent story's acceptance criteria should specify which payment
  path(s) it covers and that the unpaid/paid state stays consistent
  regardless of path — a cross-cutting NFR similar to the booking invariant.

---

## 7. Decision heuristics for prioritization

### 7.1 Value vs. Effort matrix

A simple 2×2: value on one axis, effort (implementation complexity) on the
other. Atlassian's framing of the four quadrants:

- **Do first** (high value, low effort) — quick wins.
- **Do second** (high value, high effort) — worth it, but plan capacity.
- **Do last** (low value, low effort) — fine as filler, not urgent.
- **Avoid** (low value, high effort) — usually not worth the team's time.

Atlassian's own caveat is important: "prioritization should combine
structured methods with qualitative considerations... tap into your teams'
and stakeholders' knowledge of business goals and customer needs" — the
matrix is a conversation aid, not a formula to apply blindly.

Source: [Atlassian — Prioritization frameworks](https://www.atlassian.com/agile/product-management/prioritization-framework);
[Atlassian — Prioritizing ideas for effective product development (Jira Product Discovery handbook)](https://www.atlassian.com/software/jira/product-discovery/resources/handbook/prioritization).

### 7.2 MoSCoW method

MoSCoW sorts backlog items into **M**ust have, **S**hould have, **C**ould
have, and **W**on't have (this time) — the lowercase "o"s exist only to
make the acronym pronounceable. Originally developed by **Dai Clegg** in
1994 while working at Oracle for rapid application development (RAD), and
later formalized as part of the **Dynamic Systems Development Method
(DSDM)**, an agile delivery framework, which remains its primary
professional home. It's especially useful at scope-cut time (e.g. "what
absolutely must be in this sprint/release vs. what's a should-have we can
drop under time pressure").

Source: [Wikipedia — MoSCoW method](https://en.wikipedia.org/wiki/MoSCoW_method)
(secondary, but consistent across multiple independent summaries including
[monday.com](https://monday.com/blog/project-management/moscow-prioritization-method/)
and [ProductPlan glossary](https://www.productplan.com/glossary/moscow-prioritization)).

### 7.3 Real-world Product Owner/Manager framing at scale — cross-checking against industry job descriptions

To ground this persona in how the role is actually exercised at companies
building comparable marketplace/two-sided products, not just textbook
Scrum:

- **Atlassian** postings for Product Owner roles emphasize translating
  business needs into "clear epics, features, and user stories with
  well-defined acceptance criteria," and managing platform health
  (stability, scalability, technical debt) alongside a prioritized roadmap
  — i.e., the same PO is expected to weigh feature value against
  engineering/technical-debt cost, exactly the caution-vs-value tension
  this persona is meant to arbitrate.
  [Source](https://www.atlassian.com/company/careers/details/18916).
- **Uber** Product Manager postings for marketplace roles describe owning
  "the strategy and systems that dynamically adjust... information to
  effectively manage demand," and explicitly frame the job around balancing
  two sides of a marketplace (in Uber's case riders/drivers; in this
  project, players/courts or hosts/joiners) — directly relevant to a
  court-booking marketplace with both bookers and court operators.
  [Source: Uber Careers — Senior Product Manager, Marketplace](https://jobs.uber.com/en/jobs/300215/).
- **PayPal** postings for agile Product Owner/Manager roles describe the
  job as being "customer-centric, strategic, analytical, and laser-focused
  on executing at scale," prioritizing the backlog and championing "the
  most impactful features" within a scrum team — consistent with the
  Scrum Guide's "maximizing value" accountability.
  [Source: PayPal careers listing](https://paypal.wd1.myworkdayjobs.com/en-US/jobs/job/Sr-Product-Manager---CS-Agentic-AI_R0136677).
- **Meta** Product Manager postings emphasize "defining success metrics,
  prioritizing product problems, and identifying the best strategies...
  while adapting strategy to reflect learnings" — reinforcing that
  prioritization is treated as an ongoing, evidence-revisable judgment call,
  not a one-time roadmap decision.
  [Source: Meta Careers](https://www.metacareers.com/jobs/961472602841468).

**(synthesis)** None of these postings use "Product Owner" and "Scrum" as
narrowly as the official Scrum Guide does — in industry practice the titles
blur, and a Product Owner on a small team (like this one) is usually also
doing Product Manager-shaped work (market framing, cross-team alignment).
For this persona: default to the Scrum Guide's narrow, precise
accountability (backlog value + ordering + acceptance criteria) when acting
*within* a sprint, but don't be surprised if the user also wants
Product-Manager-shaped judgment calls (positioning, sequencing across
multiple future sprints) — that is normal role-blending on small teams and
consistent with how these companies actually write the job.

---

## 8. Sources

Primary / official:
- [Scrum Guide — scrumguides.org](https://scrumguides.org/scrum-guide.html)
- [2020 Scrum Guide (US), PDF](https://scrumguides.org/docs/scrumguide/v2020/2020-Scrum-Guide-US.pdf)
- [Scrum.org — What is a Product Owner?](https://www.scrum.org/resources/what-is-a-product-owner)
- [Scrum.org — Accountabilities in Scrum: It's A Complete Picture Now](https://www.scrum.org/resources/blog/accountabilities-scrum-its-complete-picture-now)
- [Scrum.org — The Simple Difference Between the Product Owner and Product Manager](https://www.scrum.org/resources/blog/simple-difference-between-product-owner-and-product-manager)
- [Scrum Alliance — The Scrum Events](https://resources.scrumalliance.org/Article/scrum-events)
- [Scrum Alliance — Definition of Ready vs. Definition of Done](https://resources.scrumalliance.org/Article/definition-vs-ready)

Atlassian:
- [Atlassian — What is a Product Owner? Roles & Responsibilities](https://www.atlassian.com/agile/product-management/product-owner)
- [Atlassian — Product Manager: Role & Best Practices](https://www.atlassian.com/agile/product-management/product-manager)
- [Atlassian — Agile scrum roles and responsibilities](https://www.atlassian.com/agile/scrum/roles)
- [Atlassian — What is a scrum master?](https://www.atlassian.com/agile/scrum/scrum-master)
- [Atlassian — Definition of Ready](https://www.atlassian.com/agile/project-management/definition-of-ready)
- [Atlassian — Definition of Done](https://www.atlassian.com/agile/project-management/definition-of-done)
- [Atlassian — User Stories With Examples and a Template](https://www.atlassian.com/agile/project-management/user-stories)
- [Atlassian Community — Acceptance criteria in Jira: how to write, store, and validate them](https://community.atlassian.com/forums/App-Central-articles/Acceptance-criteria-in-Jira-how-to-write-store-and-validate-them/ba-p/3165137)
- [Atlassian — Prioritization frameworks](https://www.atlassian.com/agile/product-management/prioritization-framework)
- [Atlassian — Prioritizing ideas for effective product development (Jira Product Discovery handbook)](https://www.atlassian.com/software/jira/product-discovery/resources/handbook/prioritization)
- [Atlassian — Job Details (example Product Owner posting)](https://www.atlassian.com/company/careers/details/18916)

Mountain Goat Software / Mike Cohn:
- [Mountain Goat Software — User Stories](https://www.mountaingoatsoftware.com/agile/user-stories)
- [Mountain Goat Software — Agile Estimating: How Teams Estimate with Story Points](https://www.mountaingoatsoftware.com/agile/agile-estimation-estimating-with-story-points)
- [Mountain Goat Software — What Are Story Points and Why Do We Use Them?](https://www.mountaingoatsoftware.com/blog/what-are-story-points)
- [Mountain Goat Software — Don't Equate Story Points to Hours](https://www.mountaingoatsoftware.com/blog/dont-equate-story-points-to-hours)
- [Mountain Goat Software — How to Estimate Velocity as an Agile Consultant](https://www.mountaingoatsoftware.com/blog/how-to-estimate-velocity-as-an-agile-consultant)
- [Mountain Goat Software — The Best Way to Establish a Baseline When Playing Planning Poker](https://www.mountaingoatsoftware.com/blog/the-best-way-to-establish-a-baseline-when-playing-planning-poker)
- [Mountain Goat Software — About Mike Cohn](https://www.mountaingoatsoftware.com/company/about-mike-cohn)

INVEST / BDD / MoSCoW origins:
- [Agile Alliance — INVEST glossary entry (attributes to Bill Wake, 2003)](https://agilealliance.org/glossary/invest/)
- [Wikipedia — INVEST (mnemonic)](https://en.wikipedia.org/wiki/INVEST_(mnemonic))
- [Cucumber — History of BDD](https://cucumber.io/docs/bdd/history/)
- [Martin Fowler — Given-When-Then](https://martinfowler.com/bliki/GivenWhenThen.html)
- [Wikipedia — MoSCoW method](https://en.wikipedia.org/wiki/MoSCoW_method)

Estimation, corroborating/secondary:
- [AltexSoft — Story Points and Planning Poker: How to Make Estimates in Scrum](https://www.altexsoft.com/blog/story-points/)
- [Asana — Planning Poker: The All-in Strategy for Agile Estimation](https://asana.com/resources/planning-poker)

Industry job descriptions (Product Owner / Product Manager):
- [Atlassian careers — Product Owner posting](https://www.atlassian.com/company/careers/details/18916)
- [Uber Careers — Senior Product Manager, Marketplace](https://jobs.uber.com/en/jobs/300215/)
- [Uber Careers — Lead Product Manager, Mobility Marketplace](https://www.uber.com/global/en/careers/list/158040/)
- [PayPal (Workday) — Sr Product Manager posting](https://paypal.wd1.myworkdayjobs.com/en-US/jobs/job/Sr-Product-Manager---CS-Agentic-AI_R0136677)
- [Meta Careers — Product Manager posting](https://www.metacareers.com/jobs/961472602841468)

Comparative / differences (secondary, corroborating the Scrum Guide's primary framing):
- [BMC — Product Owner vs Product Manager vs Scrum Master](https://www.bmc.com/blogs/product-owner-product-manager-scrum-master/)
- [Age-of-Product — Scrum Guide 2020: Beyond Software](https://age-of-product.com/scrum-guide-2020/)

Project-internal (for domain examples in Section 6):
- `CLAUDE.md` (this repo's root project rulebook)
- `docs/reviews/01-t1-pricing-quote.md`
- `docs/reviews/03-t3-cancel-booking.md`
