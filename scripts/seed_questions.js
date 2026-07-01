import pg from 'pg';
const { Client } = pg;

const DATABASE_URL = process.env.DATABASE_URL;
if (!DATABASE_URL) {
  console.error('Error: DATABASE_URL environment variable is required');
  process.exit(1);
}

async function run() {
  const client = new Client({ connectionString: DATABASE_URL });
  await client.connect();

  const res = await client.query('SELECT count(*) FROM questions');
  console.log(`Questions count: ${res.rows[0].count}`);

  if (parseInt(res.rows[0].count) === 0) {
    await client.query(`
      INSERT INTO questions (id, question_text, question_type, options, correct_answer, difficulty, created_at, updated_at)
      VALUES (gen_random_uuid(), 'What is 2+2?', 'multiple_choice', '["3", "4", "5"]', '4', 'easy', NOW(), NOW())
    `);
    await client.query(`
      INSERT INTO questions (id, question_text, question_type, correct_answer, difficulty, created_at, updated_at)
      VALUES (gen_random_uuid(), 'Is the sky blue?', 'true_false', 'true', 'easy', NOW(), NOW())
    `);
    console.log('Inserted seed questions');
  }

  await client.end();
}

run().catch(console.error);
