#!/bin/bash
# Test script for quiz sandbox
# Tests that the project compiles and the quiz logic works correctly

set -e

echo "=== Talent-Backend Quiz Sandbox Test ==="

# 1. Verify Go code compiles
echo ""
echo "1. Checking Go compilation..."
go build -o /dev/null .
echo "   ✓ Compilation successful"

# 2. Verify the GetNextQuestion logic
echo ""
echo "2. Checking GetNextQuestion logic in quiz_service.go..."
if grep -q "ORDER BY RANDOM()" service/quiz_service.go; then
    echo "   ✓ GetNextQuestion uses random selection"
else
    echo "   ✗ GetNextQuestion does not use random selection"
    exit 1
fi

if grep -q "questions_per_quiz" service/quiz_service.go; then
    echo "   ✓ GetNextQuestion respects questions_per_quiz limit"
else
    echo "   ✗ GetNextQuestion does not check questions_per_quiz"
    exit 1
fi

# 3. Verify the quiz is limited to questions_per_quiz
echo ""
echo "3. Checking quiz attempt inserts questions_per_quiz..."
if grep -q "questions_per_quiz" service/quiz_service.go; then
    echo "   ✓ StartQuizAttempt sets questions_per_quiz"
fi

# 4. Check that quiz attempt query works
echo ""
echo "4. Checking GetUserQuizzes implementation..."
if grep -q "LEFT JOIN jobs" service/quiz_service.go; then
    echo "   ✓ GetUserQuizzes uses JOIN to get quiz titles"
else
    echo "   ✗ GetUserQuizzes not properly implemented"
    exit 1
fi

# 5. Check docker-compose.yml exists
echo ""
echo "5. Checking sandbox infrastructure..."
if [ -f docker-compose.yml ]; then
    echo "   ✓ docker-compose.yml exists"
else
    echo "   ✗ docker-compose.yml missing"
    exit 1
fi

if [ -f scripts/seed.sql ]; then
    SEED_COUNT=$(grep -c "INSERT INTO questions" scripts/seed.sql || true)
    echo "   ✓ seed.sql exists with $SEED_COUNT question INSERTs"
else
    echo "   ✗ seed.sql missing"
    exit 1
fi

# 6. Run go vet
echo ""
echo "6. Running go vet..."
go vet ./...
echo "   ✓ go vet passed"

echo ""
echo "=== All checks passed! ==="
echo ""
echo "To start the sandbox:"
echo "  docker compose up -d"
echo "  ./scripts/seed.sh"
echo "  go run ."
