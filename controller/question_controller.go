package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/Izone-hub/talent-backend/models"
	"github.com/Izone-hub/talent-backend/service"
)

// QuestionController handles HTTP requests for questions.
type QuestionController struct {
	questionService *service.QuestionService
}

// NewQuestionController creates a new QuestionController.
func NewQuestionController(questionService *service.QuestionService) *QuestionController {
	return &QuestionController{
		questionService: questionService,
	}
}

// ---------------------------------------------------------------------------
// POST /api/v1/questions — Create a new question
// ---------------------------------------------------------------------------
func (c *QuestionController) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.CreateQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	question, err := c.questionService.CreateQuestion(r.Context(), userID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, question)
}

// ---------------------------------------------------------------------------
// GET /api/v1/questions — List questions
// ---------------------------------------------------------------------------
func (c *QuestionController) ListQuestions(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	questions, err := c.questionService.ListQuestions(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"questions": questions,
		"total":     len(questions),
		"limit":     limit,
		"offset":    offset,
	})
}

// ---------------------------------------------------------------------------
// GET /api/v1/questions/{id} — Get a single question
// ---------------------------------------------------------------------------
func (c *QuestionController) GetQuestion(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid question ID")
		return
	}

	question, err := c.questionService.GetQuestion(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Question not found")
		return
	}

	writeJSON(w, http.StatusOK, question)
}

// ---------------------------------------------------------------------------
// PUT /api/v1/questions/{id} — Update a question
// ---------------------------------------------------------------------------
func (c *QuestionController) UpdateQuestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid question ID")
		return
	}

	var req models.UpdateQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	question, err := c.questionService.UpdateQuestion(r.Context(), userID, id, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, question)
}

// ---------------------------------------------------------------------------
// DELETE /api/v1/questions/{id} — Delete a question
// ---------------------------------------------------------------------------
func (c *QuestionController) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid question ID")
		return
	}

	if err := c.questionService.DeleteQuestion(r.Context(), userID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// POST /api/v1/questions/{id}/test — Test code against a question
// ---------------------------------------------------------------------------
type testQuestionRequest struct {
	Code string `json:"code"`
}

func (c *QuestionController) TestQuestion(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid question ID")
		return
	}

	var req testQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		writeError(w, http.StatusBadRequest, "Code is required")
		return
	}

	// Load question + coding details
	question, err := c.questionService.GetQuestion(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Question not found")
		return
	}

	// Only coding_challenge questions are testable
	if question.QuestionType != "coding_challenge" {
		writeError(w, http.StatusBadRequest, "Question is not a coding challenge")
		return
	}
	if question.CodingDetails == nil {
		writeError(w, http.StatusBadRequest, "No coding detail found for this question")
		return
	}

	var testCases []map[string]interface{}
	if err := json.Unmarshal(question.CodingDetails.TestCases, &testCases); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid test cases format")
		return
	}
	if len(testCases) == 0 {
		writeError(w, http.StatusBadRequest, "No test cases defined for this question")
		return
	}

	// Detect execution type:
	// 1. If test cases have "func" field → function mode
	// 2. If code defines a function → function mode
	// 3. Otherwise → standard mode
	hasFuncField := false
	for _, tc := range testCases {
		if _, ok := tc["func"]; ok {
			hasFuncField = true
			break
		}
	}

	sandbox := &service.SandboxService{}
	lang := question.CodingDetails.Language
	code := req.Code

	codeDefinesFunc := hasFunctionDefinition(code, lang)
	useFunctionMode := hasFuncField || codeDefinesFunc

	if !useFunctionMode {
		// Standard execution: run code with stdin, compare stdout to expected
		inputVal := testCases[0]["input"]
		inputBytes, _ := json.Marshal(inputVal)
		stdinVal := ""
		if string(inputBytes) != "null" {
			stdinVal = strings.Trim(string(inputBytes), "\"")
		}

		resp, err := sandbox.Execute(r.Context(), models.ExecuteRequest{
			Language: lang,
			Code:     code,
			Type:     models.ExecutionTypeStandard,
			Stdin:    stdinVal,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Execution failed: "+err.Error())
			return
		}

		// Compare stdout to expected
		expectedOut, _ := testCases[0]["expected_output"].(string)
		actualOut := strings.TrimSpace(resp.Stdout)
		expectedOut = strings.TrimSpace(expectedOut)
		passed := resp.ExitCode == 0 && actualOut == expectedOut
		resp.Passed = &passed

		writeJSON(w, http.StatusOK, resp)
		return
	}

	// Function execution: use test harness
	detectedFunc := detectFunctionName(code, lang)
	var sandboxTests []map[string]interface{}
	for _, tc := range testCases {
		fn, _ := tc["func"].(string)
		if fn == "" {
			fn = detectedFunc
		}
		args := tc["args"]
		if args == nil {
			if input, ok := tc["input"]; ok {
				args = input
			}
		}
		expected := tc["expected"]
		if expected == nil {
			expected = tc["expected_output"]
		}
		sandboxTests = append(sandboxTests, map[string]interface{}{
			"func":     fn,
			"args":     args,
			"expected": expected,
		})
	}
	sandboxTestsJSON, _ := json.Marshal(sandboxTests)

	resp, err := sandbox.Execute(r.Context(), models.ExecuteRequest{
		Language: lang,
		Code:     code,
		Type:     models.ExecutionTypeFunction,
		Stdin:    string(sandboxTestsJSON),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Execution failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// POST /api/v1/questions/{id}/validate — Validate code against ALL test cases
// ---------------------------------------------------------------------------
func (c *QuestionController) ValidateQuestion(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid question ID")
		return
	}

	var req models.ValidateQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		writeError(w, http.StatusBadRequest, "Code is required")
		return
	}

	question, err := c.questionService.GetQuestion(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Question not found")
		return
	}

	if question.QuestionType != "coding_challenge" {
		writeError(w, http.StatusBadRequest, "Question is not a coding challenge")
		return
	}
	if question.CodingDetails == nil {
		writeError(w, http.StatusBadRequest, "No coding detail found for this question")
		return
	}

	var testCases []map[string]interface{}
	if err := json.Unmarshal(question.CodingDetails.TestCases, &testCases); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid test cases format")
		return
	}
	if len(testCases) == 0 {
		writeError(w, http.StatusBadRequest, "No test cases defined for this question")
		return
	}

	lang := question.CodingDetails.Language
	code := req.Code

	hasFuncField := false
	for _, tc := range testCases {
		if _, ok := tc["func"]; ok {
			hasFuncField = true
			break
		}
	}
	codeDefinesFunc := hasFunctionDefinition(code, lang)
	useFunctionMode := hasFuncField || codeDefinesFunc

	sandbox := &service.SandboxService{}
	totalPassed := 0
	totalFailed := 0
	var results []models.TestCaseResult

	if !useFunctionMode {
		for i, tc := range testCases {
			inputVal := tc["input"]
			inputBytes, _ := json.Marshal(inputVal)
			stdinVal := ""
			if string(inputBytes) != "null" {
				stdinVal = strings.Trim(string(inputBytes), "\"")
			}

			resp, execErr := sandbox.Execute(r.Context(), models.ExecuteRequest{
				Language: lang,
				Code:     code,
				Type:     models.ExecutionTypeStandard,
				Stdin:    stdinVal,
			})
			if execErr != nil {
				writeError(w, http.StatusInternalServerError, "Execution failed: "+execErr.Error())
				return
			}

			expectedOut, _ := tc["expected_output"].(string)
			actualOut := strings.TrimSpace(resp.Stdout)
			expectedOutTrimmed := strings.TrimSpace(expectedOut)
			passed := resp.ExitCode == 0 && actualOut == expectedOutTrimmed

			errMsg := ""
			if resp.ExitCode != 0 {
				errMsg = resp.Stderr
				if errMsg == "" {
					errMsg = resp.Error
				}
			}

			isHidden, _ := tc["is_hidden"].(bool)

			result := models.TestCaseResult{
				Index:          i,
				Input:          inputVal,
				ExpectedOutput: expectedOut,
				ActualOutput:   actualOut,
				Passed:         passed,
				Error:          errMsg,
				IsHidden:       isHidden,
				TimeMs:         resp.TimeMs,
			}
			results = append(results, result)

			if passed {
				totalPassed++
			} else {
				totalFailed++
			}
		}
	} else {
		detectedFunc := detectFunctionName(code, lang)
		var sandboxTests []map[string]interface{}
		for _, tc := range testCases {
			fn, _ := tc["func"].(string)
			if fn == "" {
				fn = detectedFunc
			}
			args := tc["args"]
			if args == nil {
				if input, ok := tc["input"]; ok {
					args = input
				}
			}
			expected := tc["expected"]
			if expected == nil {
				expected = tc["expected_output"]
			}
			sandboxTests = append(sandboxTests, map[string]interface{}{
				"func":     fn,
				"args":     args,
				"expected": expected,
			})
		}
		sandboxTestsJSON, _ := json.Marshal(sandboxTests)

		resp, execErr := sandbox.Execute(r.Context(), models.ExecuteRequest{
			Language: lang,
			Code:     code,
			Type:     models.ExecutionTypeFunction,
			Stdin:    string(sandboxTestsJSON),
		})
		if execErr != nil {
			writeError(w, http.StatusInternalServerError, "Execution failed: "+execErr.Error())
			return
		}

		harnessOutput := resp.Stdout

		for i, tc := range testCases {
			expected := tc["expected"]
			if expected == nil {
				expected = tc["expected_output"]
			}
			expectedStr, _ := json.Marshal(expected)
			isHidden, _ := tc["is_hidden"].(bool)

			passLine := ""
			failLine := ""
			for _, line := range strings.Split(harnessOutput, "\n") {
				line = strings.TrimSpace(line)
				prefix := fmt.Sprintf("Test %d:", i)
				if strings.HasPrefix(line, prefix) {
					if strings.Contains(line, "PASS") {
						passLine = line
					} else {
						failLine = line
					}
				}
			}

			passed := passLine != "" && failLine == ""
			errMsg := ""
			actualOut := ""
			if passed {
				actualOut = "PASS"
			} else {
				if failLine != "" {
					errMsg = failLine
					actualOut = "FAIL"
				} else {
					errMsg = "No test output for case"
					actualOut = "UNKNOWN"
				}
			}

			_ = expectedStr

			result := models.TestCaseResult{
				Index:          i,
				Input:          tc["args"],
				ExpectedOutput: fmt.Sprintf("%v", expected),
				ActualOutput:   actualOut,
				Passed:         passed,
				Error:          errMsg,
				IsHidden:       isHidden,
				TimeMs:         resp.TimeMs,
			}
			results = append(results, result)

			if passed {
				totalPassed++
			} else {
				totalFailed++
			}
		}
	}

	resp := models.ValidateQuestionResponse{
		QuestionID:  id.String(),
		Language:    lang,
		TotalPassed: totalPassed,
		TotalFailed: totalFailed,
		TotalCases:  len(testCases),
		AllPassed:   totalFailed == 0,
		TestResults: results,
	}

	writeJSON(w, http.StatusOK, resp)
}

// hasFunctionDefinition checks if code defines a function (as opposed to just top-level statements).
func hasFunctionDefinition(code, lang string) bool {
	code = strings.TrimSpace(code)
	switch strings.ToLower(lang) {
	case "javascript":
		return strings.Contains(code, "function ") ||
			strings.Contains(code, "=>") ||
			strings.Contains(code, "const solution") ||
			strings.Contains(code, "let solution") ||
			strings.Contains(code, "var solution")
	case "python":
		return strings.Contains(code, "def ")
	case "java":
		return strings.Contains(code, "public static") || strings.Contains(code, "class ")
	case "cpp", "c++":
		return strings.Contains(code, "int solution") || strings.Contains(code, "void solution") || strings.Contains(code, "string solution")
	case "go":
		return strings.Contains(code, "func ")
	default:
		return strings.Contains(code, "function ") || strings.Contains(code, "def ") || strings.Contains(code, "func ")
	}
}

// detectFunctionName extracts the first user-defined function name from code.
func detectFunctionName(code, lang string) string {
	code = strings.TrimSpace(code)
	switch strings.ToLower(lang) {
	case "python":
		// def function_name(
		for _, line := range strings.Split(code, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "def ") {
				rest := line[4:]
				parenIdx := strings.Index(rest, "(")
				if parenIdx > 0 {
					return strings.TrimSpace(rest[:parenIdx])
				}
			}
		}
	case "javascript":
		// function function_name(  OR  const function_name = 
		for _, line := range strings.Split(code, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "function ") {
				rest := line[9:]
				parenIdx := strings.Index(rest, "(")
				if parenIdx > 0 {
					return strings.TrimSpace(rest[:parenIdx])
				}
			}
			if strings.HasPrefix(line, "const ") || strings.HasPrefix(line, "let ") || strings.HasPrefix(line, "var ") {
				equalsIdx := strings.Index(line, "=")
				if equalsIdx > 0 {
					name := strings.TrimSpace(line[6:equalsIdx])
					if name != "" {
						return name
					}
				}
			}
		}
	case "go":
		// func FunctionName(
		for _, line := range strings.Split(code, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "func ") {
				rest := line[5:]
				// Skip receiver: func (r *Type) Name(
				if strings.HasPrefix(rest, "(") {
					closeIdx := strings.Index(rest, ")")
					if closeIdx > 0 {
						rest = strings.TrimSpace(rest[closeIdx+1:])
					}
				}
				parenIdx := strings.Index(rest, "(")
				if parenIdx > 0 {
					name := strings.TrimSpace(rest[:parenIdx])
					if name != "" && name != "main" && name != "init" {
						return name
					}
				}
			}
		}
	case "java":
		// public static ReturnType FunctionName(
		for _, line := range strings.Split(code, "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "public static") || strings.Contains(line, "static public") {
				parenIdx := strings.Index(line, "(")
				if parenIdx > 0 {
					before := line[:parenIdx]
					parts := strings.Fields(before)
					if len(parts) > 0 {
						name := parts[len(parts)-1]
						if name != "" {
							return name
						}
					}
				}
			}
		}
	case "cpp", "c++":
		// Could be: int function_name(  or  string function_name(
		for _, line := range strings.Split(code, "\n") {
			line = strings.TrimSpace(line)
			parenIdx := strings.Index(line, "(")
			if parenIdx > 0 {
				before := line[:parenIdx]
				parts := strings.Fields(before)
				if len(parts) >= 2 {
					name := parts[len(parts)-1]
					// Skip common keywords
					if name != "main" && name != "if" && name != "for" && name != "while" {
						return name
					}
				}
			}
		}
	}

	// Fallback: look for any identifier before '(' that isn't a keyword
	for _, line := range strings.Split(code, "\n") {
		line = strings.TrimSpace(line)
		parenIdx := strings.Index(line, "(")
		if parenIdx > 0 {
			before := line[:parenIdx]
			parts := strings.Fields(before)
			if len(parts) > 0 {
				name := parts[len(parts)-1]
				if name != "" && name != "if" && name != "for" && name != "while" && name != "switch" {
					return name
				}
			}
		}
	}

	return "solution"
}
