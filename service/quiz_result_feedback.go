package service

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrFeedbackAlreadySubmitted = errors.New("Feedback has already been submitted for this quiz attempt and cannot be modified.")
	ErrNonEnglishText          = errors.New("Feedback must be written in English using standard Latin characters.")
	ErrProfanityDetected       = errors.New("Inappropriate or offensive language is not allowed in feedback.")
)

var (
	profanePattern       = regexp.MustCompile(`(?i)\b(f(?:uck(?:ing|er|ed|s)?|uk|ck|\*+c?k|u\*+k)|motherf(?:uck(?:ing|er|ed|s)?|\*+c?k)|sh(?:it(?:ty|ting|s)?|t|\*+t|1t)|bullsh(?:it|t|\*+t)|b(?:itch(?:es|y|ing)?|tch|1tch|\*+tch)|ass(?:es|hole|holes|hat)?|dumbass(?:es)?|jackass(?:es)?|bastard(?:s)?|cunt(?:s)?|d(?:ick(?:s|head|heads)?|1ck|\*+ck)|pussy|pussies|cock(?:s|sucker)?|whore(?:s)?|slut(?:s|ty)?|retard(?:ed|s)?|nigger(?:s)?|nigga(?:s)?|faggot(?:s)?|fag(?:s)?|wanker(?:s)?|prick(?:s)?|twat(?:s)?|douche(?:bag)?|blowjob(?:s)?)\b`)
	spacedProfanePattern = regexp.MustCompile(`(?i)\b(f[\s\.\-_*]+u[\s\.\-_*]+c[\s\.\-_*]+k|s[\s\.\-_*]+h[\s\.\-_*]+i[\s\.\-_*]+t|b[\s\.\-_*]+i[\s\.\-_*]+t[\s\.\-_*]+c[\s\.\-_*]+h|c[\s\.\-_*]+u[\s\.\-_*]+n[\s\.\-_*]+t|d[\s\.\-_*]+i[\s\.\-_*]+c[\s\.\-_*]+k)\b`)
	wordRegex            = regexp.MustCompile(`[a-zA-Z]+(?:'[a-zA-Z]+)?`)
)

var (
	dictOnce          sync.Once
	englishDictionary = make(map[string]bool)

	excludedWords = map[string]bool{
		"ere": true, "ye": true, "thou": true, "thee": true, "thy": true, "thine": true,
		"quoth": true, "yclept": true, "fain": true, "hark": true, "anon": true,
		"betwixt": true, "perchance": true, "whilom": true, "forsooth": true,
		"eft": true, "eke": true, "ern": true, "erst": true, "adz": true, "ais": true,
		"ala": true, "alb": true, "ani": true, "apo": true, "ara": true, "erg": true,
		"err": true, "asdf": true, "qwerty": true, "zxcv": true,
	}

	techAndFeedbackWords = []string{
		"ui", "ux", "api", "apis", "sdk", "sdks", "cli", "ats", "cv", "cvs", "hr", "qa", "ai", "ml", "db", "dbs",
		"sql", "nosql", "postgresql", "postgres", "mysql", "sqlite", "mongodb", "redis",
		"html", "css", "js", "ts", "jsx", "tsx", "json", "yaml", "xml", "rest", "graphql", "http", "https",
		"react", "vue", "angular", "svelte", "sveltekit", "nextjs", "nuxtjs", "node", "nodejs", "express",
		"python", "golang", "go", "rust", "kotlin", "csharp", "dotnet", "php", "ruby", "rails",
		"git", "github", "gitlab", "docker", "kubernetes", "k8s", "aws", "azure", "gcp", "linux", "bash",
		"frontend", "backend", "fullstack", "devops", "repo", "repos", "pr", "prs", "ci", "cd",
		"quiz", "quizzes", "assessment", "assessments", "score", "scores", "timer", "timers",
		"algorithm", "algorithms", "leetcode", "codewars", "syntax", "async", "await", "promise", "promises",
		"bug", "bugs", "typo", "typos", "debug", "debugging", "refactor", "refactoring",
		"feedback", "review", "reviews", "candidate", "candidates", "applicant", "applicants",
		"interview", "interviews", "job", "jobs", "salary", "salaries", "resume", "resumes",
		"dont", "doesnt", "cant", "couldnt", "wont", "wouldnt", "shouldnt", "isnt", "arent",
		"wasnt", "werent", "havent", "hasnt", "hadnt", "its", "im", "ive", "ill", "id",
		"youre", "were", "theyre", "thats", "whats", "theres", "ok", "okay", "great", "awesome",
	}
)

func hasThreeConsecutive(s string) bool {
	count := 1
	for i := 1; i < len(s); i++ {
		if s[i] == s[i-1] {
			count++
			if count >= 3 {
				return true
			}
		} else {
			count = 1
		}
	}
	return false
}

func initDict() {
	dictOnce.Do(func() {
		for _, w := range techAndFeedbackWords {
			englishDictionary[strings.ToLower(w)] = true
		}

		paths := []string{"/usr/share/dict/american-english", "/usr/share/dict/words"}
		for _, path := range paths {
			file, err := os.Open(path)
			if err == nil {
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					if line == "" || strings.Contains(line, "'") {
						continue
					}
					lower := strings.ToLower(line)
					if !excludedWords[lower] {
						englishDictionary[lower] = true
					}
				}
				file.Close()
				break
			}
		}
		for ex := range excludedWords {
			delete(englishDictionary, ex)
		}
	})
}

func FindUnreadableWords(text string) []string {
	initDict()
	tokens := wordRegex.FindAllString(text, -1)
	var unreadable []string
	for _, token := range tokens {
		clean := strings.ToLower(strings.ReplaceAll(token, "'", ""))
		if len(clean) == 1 && clean != "a" && clean != "i" {
			unreadable = append(unreadable, token)
			continue
		}
		if hasThreeConsecutive(clean) {
			unreadable = append(unreadable, token)
			continue
		}
		if excludedWords[clean] || !englishDictionary[clean] {
			unreadable = append(unreadable, token)
		}
	}
	return unreadable
}

func normalizeLeet(text string) string {
	t := strings.ToLower(text)
	t = strings.ReplaceAll(t, "@", "a")
	t = strings.ReplaceAll(t, "$", "s")
	t = strings.ReplaceAll(t, "5", "s")
	t = strings.ReplaceAll(t, "!", "i")
	t = strings.ReplaceAll(t, "1", "i")
	t = strings.ReplaceAll(t, "|", "i")
	t = strings.ReplaceAll(t, "0", "o")
	t = strings.ReplaceAll(t, "3", "e")
	return t
}

func ValidateFeedbackComment(comment string) error {
	trimmed := strings.TrimSpace(comment)
	if trimmed == "" {
		return nil
	}

	// 1. Language validation: Text must be strictly English (standard printable ASCII and whitespace)
	for _, r := range trimmed {
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		if r < 32 || r > 126 {
			return ErrNonEnglishText
		}
	}

	// 2. Profanity detection
	if profanePattern.MatchString(trimmed) || spacedProfanePattern.MatchString(trimmed) {
		return ErrProfanityDetected
	}

	norm := normalizeLeet(trimmed)
	if profanePattern.MatchString(norm) || spacedProfanePattern.MatchString(norm) {
		return ErrProfanityDetected
	}

	// 3. Unreadable or meaningless words detection (e.g. 'ere', 'asdfgh', gibberish)
	unreadable := FindUnreadableWords(trimmed)
	if len(unreadable) > 0 {
		return fmt.Errorf("Feedback contains unrecognized or meaningless word: '%s'. Please write meaningful English feedback.", unreadable[0])
	}

	return nil
}

type QuizResultFeedbackService struct {
	queries *database.Queries
	db      database.DBTX
}

func NewQuizResultFeedbackService(db database.DBTX) *QuizResultFeedbackService {
	return &QuizResultFeedbackService{queries: database.New(db), db: db}
}

type QuizResultFeedbackResponse struct {
	ID            string `json:"id"`
	UserID        string `json:"user_id"`
	QuizAttemptID string `json:"quiz_attempt_id"`
	Rating        string `json:"rating"`
	Comment       string `json:"comment"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

func quizResultFeedbackResponse(row database.QuizResultFeedback) QuizResultFeedbackResponse {
	comment := ""
	if row.Comment.Valid {
		comment = row.Comment.String
	}
	return QuizResultFeedbackResponse{
		ID:            uuid.UUID(row.ID.Bytes).String(),
		UserID:        uuid.UUID(row.UserID.Bytes).String(),
		QuizAttemptID: uuid.UUID(row.QuizAttemptID.Bytes).String(),
		Rating:        row.Rating,
		Comment:       comment,
		CreatedAt:     row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:     row.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func (s *QuizResultFeedbackService) Get(ctx context.Context, userID, quizAttemptID uuid.UUID) (*QuizResultFeedbackResponse, error) {
	item, err := s.queries.GetQuizResultFeedback(ctx, database.GetQuizResultFeedbackParams{
		UserID:        feedbackUUID(userID),
		QuizAttemptID: feedbackUUID(quizAttemptID),
	})
	if err != nil {
		return nil, err
	}
	resp := quizResultFeedbackResponse(item)
	return &resp, nil
}

func (s *QuizResultFeedbackService) GetByAttempt(ctx context.Context, quizAttemptID uuid.UUID) (*QuizResultFeedbackResponse, error) {
	items, err := s.queries.ListQuizResultFeedbackByAttempt(ctx, feedbackUUID(quizAttemptID))
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, pgx.ErrNoRows
	}
	resp := quizResultFeedbackResponse(items[0])
	return &resp, nil
}

func (s *QuizResultFeedbackService) GetByApplication(ctx context.Context, applicationID uuid.UUID) (*QuizResultFeedbackResponse, error) {
	row := s.db.QueryRow(ctx, `
		SELECT qrf.id, qrf.user_id, qrf.quiz_attempt_id, qrf.rating, qrf.comment, qrf.created_at, qrf.updated_at
		FROM quiz_result_feedback qrf
		WHERE qrf.quiz_attempt_id IN (
			SELECT id FROM quiz_attempts WHERE application_id = $1
			UNION
			SELECT quiz_id FROM job_applications WHERE id = $1 AND quiz_id IS NOT NULL
		)
		ORDER BY qrf.created_at DESC
		LIMIT 1
	`, applicationID)

	var item database.QuizResultFeedback
	err := row.Scan(
		&item.ID,
		&item.UserID,
		&item.QuizAttemptID,
		&item.Rating,
		&item.Comment,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	resp := quizResultFeedbackResponse(item)
	return &resp, nil
}

func (s *QuizResultFeedbackService) Upsert(ctx context.Context, userID, quizAttemptID uuid.UUID, rating, comment string) (*QuizResultFeedbackResponse, error) {
	// 1. Validate comment content (English only, no bad words)
	if err := ValidateFeedbackComment(comment); err != nil {
		return nil, err
	}

	// 2. Check if feedback has already been submitted (one-time feedback rule)
	existing, err := s.queries.GetQuizResultFeedback(ctx, database.GetQuizResultFeedbackParams{
		UserID:        feedbackUUID(userID),
		QuizAttemptID: feedbackUUID(quizAttemptID),
	})
	if err == nil && existing.Rating != "" {
		return nil, ErrFeedbackAlreadySubmitted
	}

	items, err := s.queries.ListQuizResultFeedbackByAttempt(ctx, feedbackUUID(quizAttemptID))
	if err == nil && len(items) > 0 {
		return nil, ErrFeedbackAlreadySubmitted
	}

	var commentVal pgtype.Text
	trimmed := strings.TrimSpace(comment)
	if trimmed != "" {
		commentVal = pgtype.Text{String: trimmed, Valid: true}
	}
	item, err := s.queries.UpsertQuizResultFeedback(ctx, database.UpsertQuizResultFeedbackParams{
		UserID:        feedbackUUID(userID),
		QuizAttemptID: feedbackUUID(quizAttemptID),
		Rating:        rating,
		Comment:       commentVal,
	})
	if err != nil {
		return nil, err
	}
	resp := quizResultFeedbackResponse(item)
	return &resp, nil
}

func (s *QuizResultFeedbackService) Delete(ctx context.Context, userID, quizAttemptID uuid.UUID) error {
	return s.queries.DeleteQuizResultFeedback(ctx, database.DeleteQuizResultFeedbackParams{
		UserID:        feedbackUUID(userID),
		QuizAttemptID: feedbackUUID(quizAttemptID),
	})
}
