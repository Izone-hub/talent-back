-- +goose Up
-- +goose StatementBegin

CREATE TYPE question_type AS ENUM (
    'multiple_choice',
    'multiple_select',
    'text',
    'true_false',
    'coding_challenge'
);

CREATE TYPE question_difficulty AS ENUM (
    'easy',
    'medium',
    'hard',
    'expert'
);

CREATE TABLE IF NOT EXISTS questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    question_text TEXT NOT NULL,
    question_type question_type NOT NULL DEFAULT 'multiple_choice',
    difficulty question_difficulty NOT NULL DEFAULT 'medium',

    -- For multiple choice questions (JSON structure)
    -- Example: [{"option": "A", "text": "Option 1", "is_correct": false}, ...]
    options JSONB,
    
    -- Correct answer (varies by type)
    -- For multiple choice: store option ID
    -- For coding: store expected output or test cases
    correct_answer TEXT,
    
    -- Explanation shown after answering
    explanation TEXT,
    
    -- Time limit in seconds (0 = no limit)
    time_limit_seconds INTEGER DEFAULT 0,
    
    -- Points awarded for correct answer
    points INTEGER DEFAULT 10,
    
    -- Tags for categorization (references tags table)
    tags TEXT[] DEFAULT '{}',
    
    -- Metadata
    created_by UUID REFERENCES users(id), -- Admin who created it
    is_active BOOLEAN DEFAULT true,
    usage_count INTEGER DEFAULT 0, -- How many times used in quizzes
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()

);

-- Coding questions specific data
CREATE TABLE IF NOT EXISTS coding_questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    
    -- Programming language (e.g., 'python', 'javascript', 'go')
    language VARCHAR(50) NOT NULL,
    
    -- Initial code template shown to user
    code_template TEXT,
    
    -- Test cases to validate solution
    -- JSON array of {input: "", expected_output: "", hidden: boolean}
    test_cases JSONB NOT NULL,
    
    -- Time limit for code execution in milliseconds
    execution_time_limit INTEGER DEFAULT 2000,
    
    -- Memory limit in MB
    memory_limit INTEGER DEFAULT 256,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Question tags junction (if using separate tags table)
CREATE TABLE IF NOT EXISTS question_tags (
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (question_id, tag_id)
);

-- Indexes
CREATE INDEX idx_questions_difficulty ON questions(difficulty);
CREATE INDEX idx_questions_type ON questions(question_type);
CREATE INDEX idx_questions_tags ON questions USING GIN(tags);
CREATE INDEX idx_questions_created_by ON questions(created_by);
CREATE INDEX idx_questions_is_active ON questions(is_active) WHERE is_active = true;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS question_tags;
DROP TABLE IF EXISTS coding_questions;
DROP TABLE IF EXISTS questions;
DROP TYPE IF EXISTS question_type;
DROP TYPE IF EXISTS question_difficulty;
-- +goose StatementEnd