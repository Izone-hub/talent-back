-- +goose Up
-- +goose StatementBegin

-- Quiz attempt status
CREATE TYPE quiz_attempt_status AS ENUM (
    'started',      
    'in_progress',   
    'paused',         
    'completed',   
    'timed_out',   
    'abandoned'       
);

-- Quiz attempts table (tracks one attempt per job application)
CREATE TABLE IF NOT EXISTS quiz_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References
    application_id UUID NOT NULL REFERENCES job_applications(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    
    -- Quiz configuration snapshot
    total_questions INTEGER NOT NULL,
    questions_per_quiz INTEGER NOT NULL,
    time_limit_minutes INTEGER, -- Total time allowed
    passing_score INTEGER NOT NULL DEFAULT 70, -- Percentage to pass
    
    -- Status tracking
    status quiz_attempt_status NOT NULL DEFAULT 'started',
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    last_activity_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Results
    score INTEGER, -- Percentage score (0-100)
    correct_answers INTEGER DEFAULT 0,
    incorrect_answers INTEGER DEFAULT 0,
    skipped_questions INTEGER DEFAULT 0,
    passed BOOLEAN,
    
    -- Time tracking
    time_spent_seconds INTEGER DEFAULT 0, -- Total active time
    auto_save_interval_seconds INTEGER DEFAULT 60, -- Auto-save every 60 seconds
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Ensure one active attempt per application
    CONSTRAINT one_active_attempt_per_application UNIQUE (application_id)
);

-- Quiz answers table (1-minute auto-saves)
CREATE TABLE IF NOT EXISTS quiz_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References
    quiz_attempt_id UUID NOT NULL REFERENCES quiz_attempts(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    
    -- User's answer
    user_answer TEXT, -- For multiple choice: option ID, for coding: code
    is_correct BOOLEAN,
    
    -- Auto-save tracking
    last_saved_at TIMESTAMP NOT NULL DEFAULT NOW(),
    save_count INTEGER DEFAULT 1, -- How many times this answer was auto-saved
    
    -- Time spent on this question
    time_spent_seconds INTEGER DEFAULT 0,
    
    -- For coding questions
    code_output TEXT, -- Output from running code
    execution_time_ms INTEGER,
    memory_used_mb FLOAT,
    
    -- Status
    is_skipped BOOLEAN DEFAULT false,
    is_reviewed BOOLEAN DEFAULT false, -- User marked for review
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- One answer per question per attempt
    UNIQUE(quiz_attempt_id, question_id)
);

-- Auto-save history (audit trail of changes)
CREATE TABLE IF NOT EXISTS quiz_answer_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_answer_id UUID NOT NULL REFERENCES quiz_answers(id) ON DELETE CASCADE,
    
    -- Snapshot of answer at save time
    user_answer TEXT,
    time_spent_seconds INTEGER,
    saved_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Metadata
    save_reason VARCHAR(50), -- 'auto', 'manual', 'timeout'
    ip_address INET,
    user_agent TEXT
);

-- Quiz results summary
CREATE TABLE IF NOT EXISTS quiz_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_attempt_id UUID NOT NULL REFERENCES quiz_attempts(id) ON DELETE CASCADE,
    
    -- Breakdown by difficulty
    easy_correct INTEGER DEFAULT 0,
    easy_total INTEGER DEFAULT 0,
    medium_correct INTEGER DEFAULT 0,
    medium_total INTEGER DEFAULT 0,
    hard_correct INTEGER DEFAULT 0,
    hard_total INTEGER DEFAULT 0,
    expert_correct INTEGER DEFAULT 0,
    expert_total INTEGER DEFAULT 0,
    
    -- Breakdown by type
    multiple_choice_correct INTEGER DEFAULT 0,
    multiple_choice_total INTEGER DEFAULT 0,
    coding_correct INTEGER DEFAULT 0,
    coding_total INTEGER DEFAULT 0,
    
    -- Time analysis
    avg_time_per_question_seconds FLOAT,
    fastest_question_time_seconds INTEGER,
    slowest_question_time_seconds INTEGER,
    
    -- AI-generated feedback (for employer only)
    ai_feedback TEXT,
    strengths TEXT[], -- Tags of areas user excelled
    weaknesses TEXT[], -- Tags of areas needing improvement
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_quiz_attempts_user_id ON quiz_attempts(user_id);
CREATE INDEX idx_quiz_attempts_application_id ON quiz_attempts(application_id);
CREATE INDEX idx_quiz_attempts_status ON quiz_attempts(status);
CREATE INDEX idx_quiz_attempts_last_activity ON quiz_attempts(last_activity_at);
CREATE INDEX idx_quiz_answers_attempt_id ON quiz_answers(quiz_attempt_id);
CREATE INDEX idx_quiz_answers_question_id ON quiz_answers(question_id);
CREATE INDEX idx_quiz_answer_history_answer_id ON quiz_answer_history(quiz_answer_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS quiz_results;
DROP TABLE IF EXISTS quiz_answer_history;
DROP TABLE IF EXISTS quiz_answers;
DROP TABLE IF EXISTS quiz_attempts;
DROP TYPE IF EXISTS quiz_attempt_status;
-- +goose StatementEnd