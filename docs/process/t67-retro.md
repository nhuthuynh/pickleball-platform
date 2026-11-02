# T67 retro

Four PRs merged: #361 (`d4c8277`) → #360 (`a8e4312`) → #359 (`0af0c63`) →
#362 (`015efea`), verified against each `merged_at`. Seven issues filed, all
open: #355, #356, #357, #358, #363, #364, #365. Three product decisions taken
and recorded in the plan's §1.

No figure below is marked for `docs-doctest`. That is deliberate and is the
first thing this retro has to say about the gate it shipped: a retro is a
historical document, and pinning it to live GitHub state would make it go red
when someone comments on a merged PR. The commands are here so a reader can
re-run them and see what changed; they are not claims about today.

## 1. The pull request about the missing review record had no review record

T66's retro finding 6 is that `HANDOFF.md` claimed "all reviewed via GitHub PR
reviews" for nine PRs carrying zero reviews. That retro shipped in **#359**.
Measured at merge:

```
$ for n in 359 360 361 362; do gh api .../pulls/$n/reviews --jq 'length'; done
#359=0   #360=2   #361=1   #362=2
```

#359 took **five review rounds** and posted none of them. Every round ran as a
direct message to a report-only reviewer and every fix was committed by the
merger, so there was never a step at which the artifact got produced.

The mechanism differed from T66's and the outcome was identical. T66's claim
came from an inherited row template; this came from a workflow with no slot for
the artifact — the same diagnosis T13 reached about issue closure (*"Attention
was not scarce; the step has no slot"*). And the sentence written into
`HANDOFF.md`'s T66 row — *"T67 posts its reviews (#360, #361, #362 each carry
one)"* — is true only because it names the three that do.

The record was written to #359 as a post-merge comment rather than as a
retroactive review, because a review posted after the merge is not what the
convention means and backdating it would have been a third instance.

**This is the sprint's primary finding because it is the one a rule could not
prevent.** `CLAUDE.md` rule 9 is 77 lines, the convention is stated in
`CLAUDE.md`'s naming section, both were read, and the artifact still was not
produced.

## 2. Across eleven review rounds, every blocking finding was a claim the code did not support

Not one behavioural bug was found in review across four PRs. The complete list:

- a compile-time proof that does not exist — `&T{}` is a valid empty composite
  literal for a struct with any number of fields. Found independently in two
  bounded contexts.
- a timeout guarantee that hangs instead of failing: 25s of silence against a
  2s limit, because `exec.CommandContext` kills only the direct child.
- a guarantee the *fix* added that nothing pinned — cutting the `WaitDelay`
  line left both new tests passing.
- a doc comment a four-line test falsifies, in seven places.
- a citation that does not support what it is cited for (three migrations
  named for a claim that needed five).
- two exit codes written from expectation rather than from running them.
- a review claim in the board of record with zero reviews behind it.

The gates caught none of these, and that is not a gate defect: all seven are
prose asserting behaviour the code does not have, and a test cannot fail on a
sentence. What caught them was adversarial readers re-running each other's
mutations — which is expensive and is not a mechanism.

## 3. A fix is where the next false claim appears

Four times a correction carried a new one, twice inside the fix for the
false-claim finding itself:

- #362's first reaping test used `sleep 47` and **passed against the unfixed
  runner** — `Run` blocked the full 47s, by which point the grandchild had
  exited and `pgrep` found nothing. A guard that cannot fail, written while
  fixing a guard that could not fail.
- #362's first stderr test wrote `printf … >&2 2>&1`; redirections apply left
  to right, so both streams landed on stderr.
- #360's fix cited three migrations for a claim whose evidence needed five.
- #359's round-2 corrections introduced two wrong exit codes, and its round-3
  correction introduced a mechanism claim (*"by reading"*) that its only
  source contradicts — entering **through** the reviewer's suggested wording,
  taken verbatim.

Both #362 cases were caught by the author running the mutation rather than
assuming it. That is the only reliable technique this sprint demonstrated.

## 4. A claim can live in an identifier, where no prose grep finds it

#360's correction reached six prose sites. The seventh was a test case
**name** — `"a registration the caller cancelled leaves the list"` — the
falsified claim restated as an identifier. Neither the reviewer's grep nor the
merger's found it; the author found it by grepping for the rule's *wording*
after changing the text the review had checked.

Its own generalisation, better than the review's: **a correction to a rule
stated in more than one place is not done until a grep for the rule's wording
comes back clean.**

This is also the site `docs-doctest` can never reach. The gate reads fenced
commands in markdown; a claim in a Go identifier is invisible to it. And a
test whose name asserts the defect reads as documentation of intended
behaviour, which is the identifier form of the three green-tests-agreeing-with-
a-defect instances T66 recorded.

## 5. The distinction worth keeping

From #362's author, after the third time that ticket's prose asserted
something nothing checked:

> A mechanism whose necessity you can argue is not the same as a mechanism
> whose removal something notices.

All three of that ticket's cases had sound arguments behind them. `WaitDelay`
was genuinely necessary — proven with a `setsid` escapee — and genuinely
unpinned. The argument is not the test.

## 6. The goal was half met, and the record says so

The sprint goal was *"a real person can see what they booked and joined."*
Both actor-scoped reads ship and are authenticated; `/v1/bookings` and
`/v1/registrations` are live. **Nothing shows them to anyone.** The Bookings
screen was cut to stay inside velocity, and `useJoinGame` still keeps its
outcome in a `ref`.

So **#353 stays open**, and the cancel half never started — its product
decision (the Host sets the refund policy per Game) arrived during the sprint
and makes that ticket 8–13 points rather than the 5 it was estimated at.

A sprint that shipped two reads and closed a visibility issue would be exactly
the board-of-record failure T66's retro opens with.

## 7. Cost, stated plainly

Three code tickets and one docs ticket took eleven review rounds, five of them
on the docs PR. Every round found something real, so none was wasted — but
"none wasted" is not the same as proportionate, and most of what the docs
rounds found was in output this session had written. An orchestrator checking
its own figures is not a check, which is why the rounds existed; that they kept
finding things says the first draft was not careful enough, not that the
process was efficient.

## Recommendations for T68

1. **Produce the review artifact as a step, not a reminder** (finding 1). The
   convention cannot be satisfied by intending to satisfy it.
2. **Derive the `HANDOFF.md` review cell from GitHub**, the way `docsindex`
   derives the rest of that file's agreement with the tree. A claim in a table
   cell has no command beside it.
3. **Grep for a rule's wording, not its file**, when correcting a rule stated
   in more than one place — and include identifiers, not only prose.
4. **#356 and #357 before any pricing form**, per decision D4. Both are
   prerequisites, and #356 is wrong today for any non-UTC client.
5. **The cancel half of #353 as its own slate**, sized for the host-configurable
   refund policy rather than the plain version.
6. **#364 is a one-line fix** and cost two implementers a misdiagnosis each.
7. Answer **#355** (is a court's booking roster public?) and **#363** (the
   default branch is 393 commits stale, so Dependabot reports 5 criticals
   against a tree nobody develops on, and `Closes #N` structurally cannot
   fire).
