-- Migration script to update application_status enum with new values
-- Run this separately if you need to add Shortlisted and InTalentPool to existing enum
-- WARNING: This will drop and recreate the enum, which requires dropping dependent objects first

-- Step 1: Drop dependent objects temporarily (if needed)
-- ALTER TABLE application ALTER COLUMN status TYPE varchar(50);
-- DROP TYPE IF EXISTS application_status CASCADE;

-- Step 2: Recreate enum with all values
-- CREATE TYPE application_status AS ENUM ('Pending', 'QuizGenerated', 'Reviewed', 'Shortlisted', 'Accepted', 'Rejected', 'InTalentPool');

-- Step 3: Restore column type
-- ALTER TABLE application ALTER COLUMN status TYPE application_status USING status::text::application_status;

-- Alternative: Add values one by one (if enum already exists)
-- ALTER TYPE application_status ADD VALUE 'Shortlisted';
-- ALTER TYPE application_status ADD VALUE 'InTalentPool';
