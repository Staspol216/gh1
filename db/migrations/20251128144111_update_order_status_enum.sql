-- +goose Up
-- +goose StatementBegin

-- 1) Convert enum columns to text so we can update values freely
ALTER TABLE orders
  ALTER COLUMN status TYPE text USING status::text;

ALTER TABLE order_records
  ALTER COLUMN status TYPE text USING status::text;

-- 2) Map old values to new labels (adjust mappings as you need)
--    Map `pending` -> `received`, `canceled` -> `storage_ended`.
UPDATE orders SET status = 'received' WHERE status = 'pending';
UPDATE orders SET status = 'storage_ended' WHERE status = 'canceled';

UPDATE order_records SET status = 'received' WHERE status = 'pending';
UPDATE order_records SET status = 'storage_ended' WHERE status = 'canceled';

-- 3) Create the new enum type with desired labels
CREATE TYPE order_status_new AS ENUM (
  'received',
  'returned',
  'delivered',
  'refunded',
  'storage_ended'
);

-- 4) Switch the columns from text to the new enum
ALTER TABLE orders
  ALTER COLUMN status TYPE order_status_new USING status::order_status_new;

ALTER TABLE order_records
  ALTER COLUMN status TYPE order_status_new USING status::order_status_new;

-- 5) Drop old enum type and rename new to the original name
DROP TYPE IF EXISTS order_status;
ALTER TYPE order_status_new RENAME TO order_status;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Reverse the steps: create old enum, convert columns back, map values back.
-- If you need a Down migration implement the reverse mapping here.
-- Example (only safe if you know how to map new values back):

CREATE TYPE order_status_old AS ENUM ('pending','delivered','refunded','returned','canceled');

ALTER TABLE orders
  ALTER COLUMN status TYPE text USING status::text;

ALTER TABLE order_records
  ALTER COLUMN status TYPE text USING status::text;

-- Map back if you want (reverse of the Up mapping)
UPDATE orders SET status = 'pending' WHERE status = 'received';
UPDATE orders SET status = 'canceled' WHERE status = 'storage_ended';

UPDATE order_records SET status = 'pending' WHERE status = 'received';
UPDATE order_records SET status = 'canceled' WHERE status = 'storage_ended';

ALTER TABLE orders
  ALTER COLUMN status TYPE order_status_old USING status::order_status_old;

ALTER TABLE order_records
  ALTER COLUMN status TYPE order_status_old USING status::order_status_old;

DROP TYPE IF EXISTS order_status;
ALTER TYPE order_status_old RENAME TO order_status;

-- +goose StatementEnd