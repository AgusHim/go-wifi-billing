-- Billing period migration/backfill for PostgreSQL.
--
-- Purpose:
-- 1. Add billing period/audit columns to bills.
-- 2. Backfill existing bills from bill_date.
-- 3. Stop if duplicate active bills exist for the same subscription-period.
-- 4. Add a unique partial index for active bills.
--
-- Run after database backup. Test on staging before production.
-- Recommended command:
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f db/migrations/20260703_billing_period_backfill_postgres.sql

BEGIN;

ALTER TABLE bills
  ADD COLUMN IF NOT EXISTS period_year integer,
  ADD COLUMN IF NOT EXISTS period_month integer,
  ADD COLUMN IF NOT EXISTS period_start timestamptz,
  ADD COLUMN IF NOT EXISTS period_end timestamptz,
  ADD COLUMN IF NOT EXISTS source varchar(50) DEFAULT 'legacy_backfill',
  ADD COLUMN IF NOT EXISTS generation_run_id uuid,
  ADD COLUMN IF NOT EXISTS paid_at timestamptz,
  ADD COLUMN IF NOT EXISTS overdue_at timestamptz,
  ADD COLUMN IF NOT EXISTS isolir_enqueued_at timestamptz,
  ADD COLUMN IF NOT EXISTS last_payment_id uuid,
  ADD COLUMN IF NOT EXISTS status_reason varchar(255);

UPDATE bills
SET
  period_year = COALESCE(period_year, EXTRACT(YEAR FROM bill_date AT TIME ZONE 'Asia/Jakarta')::integer),
  period_month = COALESCE(period_month, EXTRACT(MONTH FROM bill_date AT TIME ZONE 'Asia/Jakarta')::integer),
  period_start = COALESCE(period_start, date_trunc('month', bill_date AT TIME ZONE 'Asia/Jakarta') AT TIME ZONE 'Asia/Jakarta'),
  period_end = COALESCE(period_end, (date_trunc('month', bill_date AT TIME ZONE 'Asia/Jakarta') + interval '1 month' - interval '1 second') AT TIME ZONE 'Asia/Jakarta'),
  source = COALESCE(NULLIF(source, ''), 'legacy_backfill'),
  paid_at = CASE
    WHEN paid_at IS NULL AND lower(status) = 'paid' THEN updated_at
    ELSE paid_at
  END,
  overdue_at = CASE
    WHEN overdue_at IS NULL AND lower(status) = 'overdue' THEN updated_at
    ELSE overdue_at
  END
WHERE bill_date IS NOT NULL
  AND (
    period_year IS NULL
    OR period_month IS NULL
    OR period_start IS NULL
    OR period_end IS NULL
    OR source IS NULL
    OR source = ''
    OR (paid_at IS NULL AND lower(status) = 'paid')
    OR (overdue_at IS NULL AND lower(status) = 'overdue')
  );

COMMIT;

-- Validation 1: these rows must be fixed before automation/indexing.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM bills
    WHERE deleted_at IS NULL
      AND (
        subscription_id IS NULL
        OR period_year IS NULL
        OR period_month IS NULL
        OR period_start IS NULL
        OR period_end IS NULL
        OR period_month < 1
        OR period_month > 12
      )
  ) THEN
    RAISE EXCEPTION 'billing period migration failed: active bills still have missing/invalid period fields';
  END IF;
END $$;

-- Validation 2: duplicate active bills will break the unique index.
-- If this raises an error, run the diagnostic query below and resolve duplicates manually.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM bills
    WHERE deleted_at IS NULL
    GROUP BY subscription_id, period_year, period_month
    HAVING COUNT(*) > 1
  ) THEN
    RAISE EXCEPTION 'billing period migration stopped: duplicate active bills exist for subscription-period';
  END IF;
END $$;

-- Diagnostic query for duplicates. Run manually if Validation 2 fails.
-- SELECT
--   subscription_id,
--   period_year,
--   period_month,
--   COUNT(*) AS duplicate_count,
--   array_agg(id ORDER BY created_at ASC) AS bill_ids,
--   array_agg(public_id ORDER BY created_at ASC) AS public_ids,
--   array_agg(status ORDER BY created_at ASC) AS statuses
-- FROM bills
-- WHERE deleted_at IS NULL
-- GROUP BY subscription_id, period_year, period_month
-- HAVING COUNT(*) > 1
-- ORDER BY period_year DESC, period_month DESC, subscription_id;

-- Unique index for idempotent invoice generation.
-- CONCURRENTLY reduces write blocking, so this must stay outside BEGIN/COMMIT.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_bills_subscription_period_active
ON bills (subscription_id, period_year, period_month)
WHERE deleted_at IS NULL;

-- Helpful read index for period filters/dashboard.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_bills_period_status_active
ON bills (period_year, period_month, status)
WHERE deleted_at IS NULL;
