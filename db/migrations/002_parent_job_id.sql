ALTER TABLE jobs
ADD COLUMN IF NOT EXISTS parent_job_id TEXT;

CREATE INDEX IF NOT EXISTS jobs_parent_job_id_idx ON jobs (parent_job_id);
