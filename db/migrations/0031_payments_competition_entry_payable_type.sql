-- T61 — widen payments.payable_type's CHECK constraint to accept
-- 'competition_entry'.
--
-- THE DEFECT
--
-- `domain.PayableTypeCompetitionEntry = "competition_entry"` was added by T10.6
-- (closing #96) along with the whole Competitions payment path: the port, the
-- adapter in internal/payments/adapter/competitions, the routing in
-- app.Service, `PayableType.IsValid()`'s new case, and a cross-context
-- integration test. The one thing that change did not do is widen
-- `0005_payments.sql`'s CHECK, which still read
--
--     CHECK (payable_type IN ('booking', 'registration', 'no_show_fee'))
--
-- So the domain accepted a value Postgres refused. Every attempt to record a
-- payment for a Competition entry — online or offline — failed against a real
-- database with `SQLSTATE 23514`, violating
-- `payments_payable_type_check`. That is the Competitions payment path
-- in its entirety, broken from the sprint it was built in.
--
-- This is the same shape as the defect 0030 fixes, in the other direction:
-- there the domain and Postgres disagreed because Postgres regressed; here they
-- disagreed because Postgres was never updated. CLAUDE.md rule 4 requires both
-- halves to be kept in sync, and 22 sprints passed with them out of sync in
-- both places.
--
-- HOW IT WENT UNNOTICED
--
-- internal/payments/adapter/competitions/cross_context_integration_test.go has
-- driven this exact call since T10.6 and fails on it — but its own header said
-- "NOT EXECUTED BY ITS AUTHOR" because no Docker daemon was available, and no
-- gate on a Docker-free machine has ever executed it. Every unit-level test of
-- this path uses an in-memory Payments repository, which has no CHECK
-- constraint to violate: the fixture was more permissive than the database, so
-- it proved the routing and hid the storage. T61's first full
-- `make ci-integration` run is what surfaced it.
--
-- 0031 is the next free number: 0030 is this same sprint's weighted-capacity
-- restoration.
--
-- WHY DROP-AND-ADD RATHER THAN A SECOND CONSTRAINT
--
-- A CHECK constraint cannot be widened in place; `ADD CONSTRAINT` alongside the
-- old one would leave the narrower one still rejecting the value. The
-- constraint name is Postgres's own default for an inline column CHECK
-- (<table>_<column>_check), which is what the failing error message names, so
-- it is dropped by that name and re-added with the same name — leaving the
-- schema with one constraint, called what it was called before.
--
-- 'recurring_hire' and 'subscription' are deliberately NOT added. They are
-- rejected by `PayableType.IsValid()` too (see its own test cases), and a CHECK
-- that permits a value the domain refuses would be a different instance of the
-- same divergence this migration closes.
ALTER TABLE payments
    DROP CONSTRAINT payments_payable_type_check;

ALTER TABLE payments
    ADD CONSTRAINT payments_payable_type_check
    CHECK (payable_type IN ('booking', 'registration', 'no_show_fee', 'competition_entry'));
