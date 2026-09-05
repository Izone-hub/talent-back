-- +goose Up
-- +goose StatementBegin

-- users.acceptance_job_id is the single source of truth for "has this user
-- accepted a job?". Historically that state could only be inferred from
-- job_applications.status = 'accepted'. This migration backfills the canonical
-- field from those legacy accepted applications so no acceptance data is lost.
--
-- Only unambiguous rows are backfilled:
--   * the user does not already have an acceptance_job_id, AND
--   * the user has exactly one distinct accepted job.
--
-- Users with several accepted applications (different jobs) are left NULL on
-- purpose — they need a human decision and are surfaced by the conflict query
-- in the release notes, rather than guessing which job is correct.

WITH accepted_jobs AS (
    SELECT user_id, job_id
    FROM job_applications
    WHERE status = 'accepted'
    GROUP BY user_id, job_id
),
ranked AS (
    SELECT user_id,
           job_id,
           COUNT(*) OVER (PARTITION BY user_id) AS distinct_accepted_jobs
    FROM accepted_jobs
)
UPDATE users u
SET acceptance_job_id = r.job_id,
    updated_at = NOW()
FROM ranked r
WHERE u.id = r.user_id
  AND u.acceptance_job_id IS NULL
  AND r.distinct_accepted_jobs = 1;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- This is a data migration: the rows set here are indistinguishable from
-- acceptance records written by the live accept flow, so a Down step would
-- risk destroying real acceptance data. Intentionally irreversible.
SELECT 1;

-- +goose StatementEnd
