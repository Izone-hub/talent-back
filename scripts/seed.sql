-- Sandbox seed data for quiz testing
-- Auto-loaded by docker-compose on first startup

-- Create a sample user (GitHub-style)
INSERT INTO users (id, github_id, username, avatar_url, email, role)
VALUES
  ('11111111-1111-1111-1111-111111111111', 12345, 'testuser', 'https://avatars.githubusercontent.com/u/12345?v=4', 'testuser@example.com', 'user'),
  ('22222222-2222-2222-2222-222222222222', 67890, 'adminuser', 'https://avatars.githubusercontent.com/u/67890?v=4', 'admin@example.com', 'admin')
ON CONFLICT (id) DO NOTHING;

-- Create sample tags
INSERT INTO tags (id, name, category, description, color)
VALUES
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1', 'go',       'skill', 'Go programming language',        '#00ADD8'),
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2', 'python',   'skill', 'Python programming language',    '#3776AB'),
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3', 'javascript', 'skill', 'JavaScript programming language', '#F7DF1E'),
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa4', 'sql',      'skill', 'SQL and databases',              '#336791'),
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa5', 'docker',   'skill', 'Docker containers',              '#2496ED'),
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa6', 'svelte',   'skill', 'Svelte framework',               '#FF3E00')
ON CONFLICT (name) DO NOTHING;

-- Create sample jobs
INSERT INTO jobs (id, title, company, description, location, type, status, created_by)
VALUES
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb01', 'Backend Go Developer', 'TechCorp', 'Build scalable APIs in Go', 'Remote', 'full_time', 'published', '22222222-2222-2222-2222-222222222222'),
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb02', 'Full Stack Engineer', 'StartupInc', 'Work on web apps with JS and Python', 'Remote', 'full_time', 'published', '22222222-2222-2222-2222-222222222222'),
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb03', 'DevOps Engineer', 'CloudBase', 'Container orchestration and infra', 'On-site', 'full_time', 'published', '22222222-2222-2222-2222-222222222222')
ON CONFLICT (id) DO NOTHING;

-- Link jobs to tags
INSERT INTO job_tags (job_id, tag_id)
VALUES
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb01', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1'),
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb01', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa4'),
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb02', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2'),
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb02', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3'),
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb03', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa5'),
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb03', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa4')
ON CONFLICT DO NOTHING;

-- 15 sample questions across different tags and difficulties
INSERT INTO questions (id, question_text, question_type, difficulty, options, correct_answer, explanation, points, tags, created_by)
VALUES
  -- Go questions (5)
  ('cccccccc-cccc-cccc-cccc-cccccccccc01',
   'What keyword is used to define a function in Go?',
   'multiple_choice', 'easy',
   '[{"option":"A","text":"func"},{"option":"B","text":"function"},{"option":"C","text":"def"},{"option":"D","text":"fn"}]',
   'A', 'func is the only correct keyword for function declaration in Go.', 10,
   '{go}',
   '22222222-2222-2222-2222-222222222222'),

  ('cccccccc-cccc-cccc-cccc-cccccccccc02',
   'What is the zero value of a pointer in Go?',
   'multiple_choice', 'easy',
   '[{"option":"A","text":"0"},{"option":"B","text":"nil"},{"option":"C","text":"undefined"},{"option":"D","text":"null"}]',
   'B', 'Pointers in Go have a zero value of nil.', 10,
   '{go}',
   '22222222-2222-2222-2222-222222222222'),

  ('cccccccc-cccc-cccc-cccc-cccccccccc03',
   'Which of the following is a valid way to declare a slice in Go?',
   'multiple_choice', 'medium',
   '[{"option":"A","text":"var s []int"},{"option":"B","text":"s := []int{}"},{"option":"C","text":"s := make([]int, 0)"},{"option":"D","text":"All of the above"}]',
   'D', 'All three syntaxes are valid ways to declare a slice in Go.', 15,
   '{go}',
   '22222222-2222-2222-2222-222222222222'),

  ('cccccccc-cccc-cccc-cccc-cccccccccc04',
   'What does the `defer` keyword do in Go?',
   'multiple_choice', 'medium',
   '[{"option":"A","text":"Pauses execution"},{"option":"B","text":"Schedules a function call to run after the surrounding function returns"},{"option":"C","text":"Creates a new goroutine"},{"option":"D","text":"Defines a constant"}]',
   'B', 'defer schedules a function call to run after the function completes.', 15,
   '{go}',
   '22222222-2222-2222-2222-222222222222'),

  ('cccccccc-cccc-cccc-cccc-cccccccccc05',
   'In Go, which type of channel is unbuffered by default?',
   'multiple_choice', 'hard',
   '[{"option":"A","text":"Read-only channels"},{"option":"B","text":"Write-only channels"},{"option":"C","text":"Bidirectional channels"},{"option":"D","text":"All channels are unbuffered by default"}]',
   'C', 'Bidirectional channels created with make(chan T) are unbuffered by default.', 20,
   '{go}',
   '22222222-2222-2222-2222-222222222222'),

  -- Python questions (3)
  ('cccccccc-cccc-cccc-cccc-cccccccccc06',
   'Which of the following is a mutable data type in Python?',
   'multiple_choice', 'easy',
   '[{"option":"A","text":"tuple"},{"option":"B","text":"list"},{"option":"C","text":"string"},{"option":"D","text":"frozenset"}]',
   'B', 'Lists are mutable; tuples, strings, and frozensets are immutable.', 10,
   '{python}',
   '22222222-2222-2222-2222-222222222222'),

  ('cccccccc-cccc-cccc-cccc-cccccccccc07',
   'What does the `range()` function return in Python 3?',
   'multiple_choice', 'medium',
   '[{"option":"A","text":"A list"},{"option":"B","text":"A tuple"},{"option":"C","text":"A range object"},{"option":"D","text":"A generator"}]',
   'C', 'range() returns a range object (an iterable), not a list.', 15,
   '{python}',
   '22222222-2222-2222-2222-222222222222'),

  ('cccccccc-cccc-cccc-cccc-cccccccccc08',
   'What is a decorator in Python?',
   'multiple_choice', 'hard',
   '[{"option":"A","text":"A design pattern for classes"},{"option":"B","text":"A function that takes another function and extends its behavior"},{"option":"C","text":"A way to declare constants"},{"option":"D","text":"An annotation for type hints"}]',
   'B', 'A decorator is a function that wraps another function to modify its behavior.', 20,
   '{python}',
   '22222222-2222-2222-2222-222222222222'),

  -- SQL questions (3)
  ('cccccccc-cccc-cccc-cccc-cccccccccc09',
   'Which SQL clause is used to filter rows after aggregation?',
   'multiple_choice', 'medium',
   '[{"option":"A","text":"WHERE"},{"option":"B","text":"HAVING"},{"option":"C","text":"FILTER"},{"option":"D","text":"BEFORE"}]',
   'B', 'HAVING is used to filter groups after aggregation; WHERE filters before.', 15,
   '{sql}',
   '22222222-2222-2222-2222-222222222222'),

  ('cccccccc-cccc-cccc-cccc-cccccccccc10',
   'What type of JOIN returns all rows from the left table and matching rows from the right?',
   'multiple_choice', 'easy',
   '[{"option":"A","text":"INNER JOIN"},{"option":"B","text":"RIGHT JOIN"},{"option":"C","text":"LEFT JOIN"},{"option":"D","text":"FULL OUTER JOIN"}]',
   'C', 'LEFT JOIN (or LEFT OUTER JOIN) returns all rows from the left table.', 10,
   '{sql}',
   '22222222-2222-2222-2222-222222222222'),

  ('cccccccc-cccc-cccc-cccc-cccccccccc11',
   'Which PostgreSQL data type is best for storing a UUID?',
   'multiple_choice', 'medium',
   '[{"option":"A","text":"VARCHAR(36)"},{"option":"B","text":"TEXT"},{"option":"C","text":"UUID"},{"option":"D","text":"CHAR(32)"}]',
   'C', 'PostgreSQL has a native UUID data type for storing UUIDs efficiently.', 15,
   '{sql}',
   '22222222-2222-2222-2222-222222222222'),

  -- JavaScript questions (2)
  ('cccccccc-cccc-cccc-cccc-cccccccccc12',
   'What does `typeof null` return in JavaScript?',
   'multiple_choice', 'medium',
   '[{"option":"A","text":"null"},{"option":"B","text":"undefined"},{"option":"C","text":"object"},{"option":"D","text":"boolean"}]',
   'C', 'typeof null returns "object" — a well-known JavaScript quirk.', 15,
   '{javascript}',
   '22222222-2222-2222-2222-222222222222'),

  ('cccccccc-cccc-cccc-cccc-cccccccccc13',
   'Which method creates a new array with the results of calling a function on every element?',
   'multiple_choice', 'easy',
   '[{"option":"A","text":"forEach()"},{"option":"B","text":"map()"},{"option":"C","text":"filter()"},{"option":"D","text":"reduce()"}]',
   'B', 'map() creates a new array by applying a function to each element.', 10,
   '{javascript}',
   '22222222-2222-2222-2222-222222222222'),

  -- Docker questions (2)
  ('cccccccc-cccc-cccc-cccc-cccccccccc14',
   'What is the purpose of a Dockerfile?',
   'multiple_choice', 'easy',
   '[{"option":"A","text":"Run containers"},{"option":"B","text":"Build container images"},{"option":"C","text":"Push images to registry"},{"option":"D","text":"Manage container networks"}]',
   'B', 'A Dockerfile contains instructions to build a Docker image.', 10,
   '{docker}',
   '22222222-2222-2222-2222-222222222222'),

  ('cccccccc-cccc-cccc-cccc-cccccccccc15',
   'Which command builds a Docker image from a Dockerfile?',
   'multiple_choice', 'easy',
   '[{"option":"A","text":"docker run"},{"option":"B","text":"docker create"},{"option":"C","text":"docker build"},{"option":"D","text":"docker compose"}]',
   'C', 'docker build reads a Dockerfile and produces a container image.', 10,
   '{docker}',
   '22222222-2222-2222-2222-222222222222')
ON CONFLICT (id) DO NOTHING;

-- Link questions to tags in the junction table
INSERT INTO question_tags (question_id, tag_id)
VALUES
  ('cccccccc-cccc-cccc-cccc-cccccccccc01', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1'),
  ('cccccccc-cccc-cccc-cccc-cccccccccc02', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1'),
  ('cccccccc-cccc-cccc-cccc-cccccccccc03', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1'),
  ('cccccccc-cccc-cccc-cccc-cccccccccc04', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1'),
  ('cccccccc-cccc-cccc-cccc-cccccccccc05', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1'),
  ('cccccccc-cccc-cccc-cccc-cccccccccc06', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2'),
  ('cccccccc-cccc-cccc-cccc-cccccccccc07', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2'),
  ('cccccccc-cccc-cccc-cccc-cccccccccc08', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2'),
  ('cccccccc-cccc-cccc-cccc-cccccccccc09', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa4'),
  ('cccccccc-cccc-cccc-cccc-cccccccccc10', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa4'),
  ('cccccccc-cccc-cccc-cccc-cccccccccc11', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa4'),
  ('cccccccc-cccc-cccc-cccc-cccccccccc12', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3'),
  ('cccccccc-cccc-cccc-cccc-cccccccccc13', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3'),
  ('cccccccc-cccc-cccc-cccc-cccccccccc14', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa5'),
  ('cccccccc-cccc-cccc-cccc-cccccccccc15', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa5')
ON CONFLICT DO NOTHING;

-- Create a sample application for the test user
INSERT INTO job_applications (id, user_id, job_id, status)
VALUES
  ('dddddddd-dddd-dddd-dddd-dddddddddd01', '11111111-1111-1111-1111-111111111111', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb01', 'submitted')
ON CONFLICT (id) DO NOTHING;
