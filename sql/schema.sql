-- Enums (Create only if they don't exist)
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'job_roles') THEN
        CREATE TYPE job_roles AS ENUM ('Frontend', 'Backend', 'Fullstack', 'Ui_Ux');
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'job_status') THEN
        CREATE TYPE job_status AS ENUM ('Open', 'Closed');
    END IF;
END $$;

-- Handle application_status enum - create if doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'application_status') THEN
        CREATE TYPE application_status AS ENUM ('Pending', 'QuizGenerated', 'Reviewed', 'Shortlisted', 'Accepted', 'Rejected', 'InTalentPool');
    END IF;
END $$;

-- Users Table (Supports both Admin and Developer)
-- Create table with basic structure first
CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY,
    first_name varchar(255) NOT NULL,
    last_name varchar(255) NOT NULL,
    email varchar(255) NOT NULL UNIQUE,
    password varchar(255) NOT NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NOT NULL DEFAULT now()
);

-- Add new columns to users table if they don't exist
DO $$ 
BEGIN
    -- Make password nullable if it's not already
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'password' AND is_nullable = 'NO') THEN
        ALTER TABLE users ALTER COLUMN password DROP NOT NULL;
    END IF;
    
    -- Add GitHub OAuth columns
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'github_username') THEN
        ALTER TABLE users ADD COLUMN github_username varchar(255) UNIQUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'github_id') THEN
        ALTER TABLE users ADD COLUMN github_id bigint UNIQUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'avatar_url') THEN
        ALTER TABLE users ADD COLUMN avatar_url varchar(500);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'github_access_token') THEN
        ALTER TABLE users ADD COLUMN github_access_token text;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'bio') THEN
        ALTER TABLE users ADD COLUMN bio text;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'location') THEN
        ALTER TABLE users ADD COLUMN location varchar(255);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'company') THEN
        ALTER TABLE users ADD COLUMN company varchar(255);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'blog') THEN
        ALTER TABLE users ADD COLUMN blog varchar(255);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'tech_stack') THEN
        ALTER TABLE users ADD COLUMN tech_stack jsonb;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'talent_status') THEN
        ALTER TABLE users ADD COLUMN talent_status varchar(50) DEFAULT 'Active';
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'availability_status') THEN
        ALTER TABLE users ADD COLUMN availability_status varchar(50) DEFAULT 'Available';
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'experience_level') THEN
        ALTER TABLE users ADD COLUMN experience_level varchar(50);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'role') THEN
        ALTER TABLE users ADD COLUMN role varchar(50) DEFAULT 'applicant';
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'auth_provider') THEN
        ALTER TABLE users ADD COLUMN auth_provider varchar(50);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'last_active_at') THEN
        ALTER TABLE users ADD COLUMN last_active_at timestamp;
    END IF;
END $$;

-- Indexes for users
CREATE INDEX IF NOT EXISTS idx_users_github_id ON users(github_id);
CREATE INDEX IF NOT EXISTS idx_users_github_username ON users(github_username);
CREATE INDEX IF NOT EXISTS idx_users_talent_status ON users(talent_status);
CREATE INDEX IF NOT EXISTS idx_users_availability ON users(availability_status);
CREATE INDEX IF NOT EXISTS idx_users_tech_stack_gin ON users USING GIN(tech_stack);
CREATE INDEX IF NOT EXISTS idx_users_experience ON users(experience_level);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Job Categories Table
CREATE TABLE IF NOT EXISTS job_categories (
    id uuid PRIMARY KEY,
    name varchar(255) UNIQUE NOT NULL,
    description text,
    created_at timestamp NOT NULL DEFAULT now()
);

-- Jobs Table
CREATE TABLE IF NOT EXISTS jobs (
    id uuid PRIMARY KEY,
    title varchar(255) NOT NULL,
    description text NOT NULL,
    role job_roles NOT NULL,
    category_id uuid NOT NULL,
    requirements text,
    location varchar(255),
    job_type varchar(50), -- Full-time, Part-time, Contract, Remote
    status job_status NOT NULL DEFAULT 'Open',
    created_by uuid NOT NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    FOREIGN KEY (category_id) REFERENCES job_categories(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

-- Applicants Table
CREATE TABLE IF NOT EXISTS applicants (
    id uuid PRIMARY KEY,
    first_name varchar(255) NOT NULL,
    last_name varchar(255) NOT NULL,
    email varchar(255) UNIQUE NOT NULL,
    github_link varchar(255) UNIQUE,
    linkedin_link varchar(255) UNIQUE,
    resume_pdf varchar(500) NOT NULL, -- file path or URL
    created_at timestamp NOT NULL DEFAULT now()
);

-- Repositories Table
CREATE TABLE IF NOT EXISTS repositories (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    github_repo_id bigint UNIQUE NOT NULL,
    name varchar(255) NOT NULL,
    full_name varchar(500) NOT NULL,
    description text,
    url varchar(500) NOT NULL,
    html_url varchar(500) NOT NULL,
    language varchar(100),
    languages jsonb,
    readme_content text,
    readme_html varchar(500),
    tech_stack jsonb,
    stars integer DEFAULT 0,
    forks integer DEFAULT 0,
    is_private boolean DEFAULT false,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NOT NULL DEFAULT now(),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_repositories_user_id ON repositories(user_id);
CREATE INDEX IF NOT EXISTS idx_repositories_language ON repositories(language);
CREATE INDEX IF NOT EXISTS idx_repositories_tech_stack ON repositories USING GIN(tech_stack);

-- Talent Pool Table
CREATE TABLE IF NOT EXISTS talent_pool (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL UNIQUE,
    status varchar(50) DEFAULT 'Active',
    tags text[],
    notes text,
    rating integer CHECK (rating >= 1 AND rating <= 5),
    last_contacted_at timestamp,
    last_applied_at timestamp,
    total_applications integer DEFAULT 0,
    accepted_jobs integer DEFAULT 0,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NOT NULL DEFAULT now(),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_talent_pool_status ON talent_pool(status);
CREATE INDEX IF NOT EXISTS idx_talent_pool_tags ON talent_pool USING GIN(tags);

-- Application Table (Updated to reference users instead of applicants)
CREATE TABLE IF NOT EXISTS application (
    id uuid PRIMARY KEY,
    applicant_id uuid NOT NULL,
    job_id uuid NOT NULL,
    status application_status NOT NULL DEFAULT 'Pending',
    generated_quiz uuid,
    applied_at timestamp NOT NULL DEFAULT now(),
    FOREIGN KEY (applicant_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

-- Quiz Table
CREATE TABLE IF NOT EXISTS quiz (
    id uuid PRIMARY KEY,
    application_id uuid NOT NULL,
    questions jsonb NOT NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE
);