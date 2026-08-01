-- Seed pricing rules for Court 1, for local smoke testing (README.md).
INSERT INTO pricing_rules (court_id, band, weekdays, start_min, end_min, price_cents) VALUES
    ('11111111-1111-1111-1111-111111111111', 'weekday', ARRAY[1,2,3,4,5]::smallint[], 360, 1020, 2000),
    ('11111111-1111-1111-1111-111111111111', 'peak',    ARRAY[1,2,3,4,5]::smallint[], 1020, 1320, 3500),
    ('11111111-1111-1111-1111-111111111111', 'weekend', ARRAY[0,6]::smallint[], 360, 1320, 3000);
