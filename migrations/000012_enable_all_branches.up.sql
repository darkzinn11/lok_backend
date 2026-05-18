-- Ensure Sede São Luís is active and correct
INSERT INTO branches (id, name, city, uf, status)
VALUES ('11111111-1111-1111-1111-111111111111', 'Sede São Luís', 'São Luís', 'MA', 'ACTIVE')
ON CONFLICT (id) DO UPDATE 
SET name = 'Sede São Luís', city = 'São Luís', uf = 'MA', status = 'ACTIVE';

-- Delete any other duplicate names if they exist to prevent unique conflict
DELETE FROM branches WHERE name = 'Belém/PA' AND id <> '22222222-2222-2222-2222-222222222222';
INSERT INTO branches (id, name, city, uf, status)
VALUES ('22222222-2222-2222-2222-222222222222', 'Belém/PA', 'Belém', 'PA', 'ACTIVE')
ON CONFLICT (id) DO UPDATE 
SET name = 'Belém/PA', city = 'Belém', uf = 'PA', status = 'ACTIVE';

DELETE FROM branches WHERE name = 'Marabá/PA' AND id <> '33333333-3333-3333-3333-333333333333';
INSERT INTO branches (id, name, city, uf, status)
VALUES ('33333333-3333-3333-3333-333333333333', 'Marabá/PA', 'Marabá', 'PA', 'ACTIVE')
ON CONFLICT (id) DO UPDATE 
SET name = 'Marabá/PA', city = 'Marabá', uf = 'PA', status = 'ACTIVE';

DELETE FROM branches WHERE name = 'Teresina/PI' AND id <> '44444444-4444-4444-4444-444444444444';
INSERT INTO branches (id, name, city, uf, status)
VALUES ('44444444-4444-4444-4444-444444444444', 'Teresina/PI', 'Teresina', 'PI', 'ACTIVE')
ON CONFLICT (id) DO UPDATE 
SET name = 'Teresina/PI', city = 'Teresina', uf = 'PI', status = 'ACTIVE';

DELETE FROM branches WHERE name = 'Recife/PE' AND id <> '55555555-5555-5555-5555-555555555555';
INSERT INTO branches (id, name, city, uf, status)
VALUES ('55555555-5555-5555-5555-555555555555', 'Recife/PE', 'Recife', 'PE', 'ACTIVE')
ON CONFLICT (id) DO UPDATE 
SET name = 'Recife/PE', city = 'Recife', uf = 'PE', status = 'ACTIVE';
