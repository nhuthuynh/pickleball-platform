-- Game entry fee (T9.2, docs/process/t9-sprint-plan.md T9.2 section).
--
-- T8.10 shipped the Join-a-Game payment flow against a price that did not
-- exist: domain.Game had no fee field at all, so the web client invented a
-- flat PLACEHOLDER_REGISTRATION_FEE_CENTS ($10.00) and labelled it
-- "placeholder" everywhere it appeared. That was accepted only as a
-- *disclosed* workaround, with an explicit follow-up recommendation to add
-- a real per-Game price the same way T8.6/T8.7 added PaymentMethod. This
-- migration is the DB-side half of that follow-up; the domain-side half is
-- domain.Game.EntryFee / domain.Money (internal/socialplay/domain).
--
-- Two columns, not one, per ADR-0005 (docs/adr/0005-currency-code-column.md):
-- every monetary amount carries an explicit ISO 4217 currency code
-- alongside its integer-minor-units amount, from day one, even while v1
-- only ever populates the code with the single launch-market value. The
-- amount/currency pair mirrors the existing pricing_rules and payments
-- columns exactly -- this migration follows that established precedent
-- rather than introducing a new convention.
--
-- Why 0 and 'USD' are the only safe defaults for pre-existing rows
-- --------------------------------------------------------------------
-- entry_fee_cents DEFAULT 0 means FREE, and that is a deliberate,
-- conservative choice rather than a neutral-looking placeholder:
--
--   * Every games row that already exists was created before a Host could
--     express a price at all. No Host ever set a fee on any of them, so
--     there is no correct non-zero figure to backfill -- and backfilling
--     one anyway (say, the retired $10.00 placeholder) would INVENT A
--     CHARGE NO HOST EVER SET, against players who joined a Game whose
--     listing never showed them a price. That is a money-correctness bug
--     of the worst kind: it is silent, it favours the platform, and it
--     would be discovered by a player being charged, not by a test.
--   * 0 is also the safe direction of the two possible errors. If 0 is
--     wrong for some historical Game, the failure mode is "a Host has to
--     set their price", which is visible and self-correcting. If a non-zero
--     backfill is wrong, the failure mode is "a player was overcharged",
--     which is neither.
--   * 0 is a REAL VALUE, not a sentinel. It means a free Game, a legitimate
--     product state the domain accepts (domain.Money.IsFree) and the UI
--     renders as the word "Free". Nothing downstream needs to distinguish
--     "free" from "unset", which is precisely why "unset" does not need to
--     be representable.
--
-- entry_fee_currency DEFAULT 'USD' follows ADR-0005's existing
-- currency-column precedent (the launch market's single currency, the same
-- constant the pricing_rules and payments columns carry). Note this means
-- backfilled rows read back as Money{0, 'USD'} while a Host who sets no fee
-- through the API can also produce Money{0, ''} -- both satisfy IsFree and
-- are treated identically everywhere downstream, so the two spellings of
-- "free" are not a distinction the application has to care about (see
-- domain.Money.Validate's doc comment, which spells this out).
--
-- The CHECK is the Postgres half of CLAUDE.md rule 4's dual enforcement:
-- domain.Money.Validate rejects a negative amount in Go, and this
-- constraint makes it unrepresentable in the database. Both must be kept in
-- sync if either ever changes. A negative entry fee would mean a Game that
-- pays players to attend, which is not a product this platform has.
--
-- New rows always supply both columns explicitly from the adapter (see
-- db/queries/socialplay.sql's CreateGame doc comment); the DEFAULTs exist
-- only to backfill rows that predate these columns.

ALTER TABLE games
    ADD COLUMN entry_fee_cents bigint NOT NULL DEFAULT 0,
    ADD COLUMN entry_fee_currency text NOT NULL DEFAULT 'USD';

ALTER TABLE games
    ADD CONSTRAINT games_entry_fee_cents_non_negative CHECK (entry_fee_cents >= 0);
