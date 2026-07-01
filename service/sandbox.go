package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Izone-hub/talent-backend/models"
)

type SandboxService struct{}

func NewSandboxService() *SandboxService {
	return &SandboxService{}
}

type langConfig struct {
	Image    string
	Ext      string
	Filename string
	RunCmd   []string // template; {file} is replaced
	TestCmd  []string
	Env      []string
}

func (s *SandboxService) langConfig(lang string) (langConfig, error) {
	switch strings.ToLower(lang) {
	case "python", "py":
		return langConfig{
			Image:   "python:3.12-slim",
			Ext:     ".py",
			RunCmd:  []string{"python", "{file}"},
			TestCmd: []string{"python", "-m", "pytest", "{file}", "-v"},
		}, nil
	case "javascript", "js", "node":
		return langConfig{
			Image:   "node:22-slim",
			Ext:     ".js",
			RunCmd:  []string{"node", "{file}"},
			TestCmd: []string{"npx", "jest", "{file}", "--verbose"},
		}, nil
	case "go", "golang":
		return langConfig{
			Image:   "sandbox-go:1.25",
			Ext:     ".go",
			RunCmd:  []string{"sh", "-c", "cd /code && go build -o /tmp/prog . && /tmp/prog"},
			TestCmd: []string{"sh", "-c", "go test -v /code/"},
			Env:     []string{"GO111MODULE=off", "GOCACHE=/root/.cache/go-build", "GOPATH=/tmp/go"},
		}, nil
	case "java":
		return langConfig{
			Image:    "eclipse-temurin:23-jdk-alpine",
			Ext:      ".java",
			Filename: "Main.java",
			RunCmd:   []string{"sh", "-c", "javac -d /tmp /code/Main.java && java -cp /tmp Main"},
			TestCmd:  []string{"java", "-jar", "{file}"},
		}, nil
	case "cpp", "c++":
		return langConfig{
			Image:   "gcc:14-bookworm",
			Ext:     ".cpp",
			RunCmd:  []string{"sh", "-c", "g++ -o /tmp/prog {file} && /tmp/prog"},
			TestCmd: []string{"sh", "-c", "g++ -o /tmp/test {file} && /tmp/test"},
		}, nil
	case "c":
		return langConfig{
			Image:   "gcc:14-bookworm",
			Ext:     ".c",
			RunCmd:  []string{"sh", "-c", "gcc -o /tmp/prog {file} && /tmp/prog"},
			TestCmd: []string{"sh", "-c", "gcc -o /tmp/test {file} && /tmp/test"},
		}, nil
	case "rust", "rs":
		return langConfig{
			Image:   "rust:1.85-slim-bookworm",
			Ext:     ".rs",
			RunCmd:  []string{"sh", "-c", "rustc -o /tmp/prog {file} && /tmp/prog"},
			TestCmd: []string{"cargo", "test"},
		}, nil
	case "ruby", "rb":
		return langConfig{
			Image:   "ruby:3.4-slim",
			Ext:     ".rb",
			RunCmd:  []string{"ruby", "{file}"},
			TestCmd: []string{"ruby", "-r", "minitest", "{file}"},
		}, nil
	case "typescript", "ts":
		return langConfig{
			Image:   "sandbox-node:22",
			Ext:     ".ts",
			RunCmd:  []string{"npx", "tsx", "{file}"},
			TestCmd: []string{"npx", "vitest", "run", "{file}"},
		}, nil
	default:
		return langConfig{}, fmt.Errorf("unsupported language: %s", lang)
	}
}

func (s *SandboxService) Execute(ctx context.Context, req models.ExecuteRequest) (*models.ExecuteResponse, error) {
	cfg, err := s.langConfig(req.Language)
	if err != nil {
		return &models.ExecuteResponse{Error: err.Error()}, nil
	}

	switch req.Type {
	case "", models.ExecutionTypeStandard:
		return s.executeStandard(ctx, cfg, req)
	case models.ExecutionTypeFunction:
		return s.executeFunction(ctx, cfg, req)
	case models.ExecutionTypeFramework:
		return s.executeFramework(ctx, cfg, req)
	default:
		return &models.ExecuteResponse{Error: fmt.Sprintf("unsupported execution type: %s", req.Type)}, nil
	}
}

func (s *SandboxService) executeStandard(ctx context.Context, cfg langConfig, req models.ExecuteRequest) (*models.ExecuteResponse, error) {
	workDir, err := os.MkdirTemp("", "sandbox-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create work dir: %w", err)
	}
	defer os.RemoveAll(workDir)
	os.Chmod(workDir, 0755)

	filename := cfg.Filename
	if filename == "" {
		filename = "main" + cfg.Ext
	}
	codeFile := filepath.Join(workDir, filename)
	if err := os.WriteFile(codeFile, []byte(req.Code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write code: %w", err)
	}

	cmdArgs := replaceFile(cfg.RunCmd, "/code/"+filename)
	return s.runDocker(ctx, cfg.Image, workDir, cmdArgs, req.Stdin, req.TimeLimit, req.MemoryLimit, cfg.Env...)
}

func (s *SandboxService) executeFunction(ctx context.Context, cfg langConfig, req models.ExecuteRequest) (*models.ExecuteResponse, error) {
	workDir, err := os.MkdirTemp("", "sandbox-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create work dir: %w", err)
	}
	defer os.RemoveAll(workDir)
	os.Chmod(workDir, 0755)

	codeFile := filepath.Join(workDir, "solution"+cfg.Ext)
	if err := os.WriteFile(codeFile, []byte(req.Code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write code: %w", err)
	}

	testHarness, err := s.generateTestHarness(req.Language, req.Stdin)
	if err != nil {
		return &models.ExecuteResponse{Error: err.Error()}, nil
	}
	testFile := filepath.Join(workDir, "test_runner"+cfg.Ext)
	if err := os.WriteFile(testFile, []byte(testHarness), 0644); err != nil {
		return nil, fmt.Errorf("failed to write test harness: %w", err)
	}

	cmdArgs := replaceFile(cfg.RunCmd, "/code/test_runner"+cfg.Ext)
	resp, dErr := s.runDocker(ctx, cfg.Image, workDir, cmdArgs, "", req.TimeLimit, req.MemoryLimit, cfg.Env...)
	if dErr != nil {
		return nil, dErr
	}

	if resp.ExitCode == 0 {
		passed := true
		resp.Passed = &passed
	} else {
		passed := false
		resp.Passed = &passed
	}
	return resp, nil
}

func (s *SandboxService) executeFramework(ctx context.Context, cfg langConfig, req models.ExecuteRequest) (*models.ExecuteResponse, error) {
	workDir, err := os.MkdirTemp("", "sandbox-framework-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create work dir: %w", err)
	}
	defer os.RemoveAll(workDir)
	os.Chmod(workDir, 0755)

	if err := s.copyTemplate(workDir, req.TemplateID, req.Language); err != nil {
		return nil, fmt.Errorf("failed to copy template: %w", err)
	}

	for filePath, content := range req.Files {
		fullPath := filepath.Join(workDir, filePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create dir for %s: %w", filePath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", filePath, err)
		}
	}

	testCmd := s.buildFrameworkTestCmd(req.Language)
	resp, dErr := s.runDocker(ctx, cfg.Image, workDir, testCmd, "", req.TimeLimit, req.MemoryLimit, cfg.Env...)
	if dErr != nil {
		return nil, dErr
	}

	if resp.ExitCode == 0 {
		passed := true
		resp.Passed = &passed
	} else {
		passed := false
		resp.Passed = &passed
	}
	return resp, nil
}

func (s *SandboxService) runDocker(ctx context.Context, image, workDir string, cmdArgs []string, stdin string, timeLimit, memLimit int, envVars ...string) (*models.ExecuteResponse, error) {
	dockerArgs := []string{"run", "--rm", "--network", "none"}

	for _, e := range envVars {
		if e != "" {
			dockerArgs = append(dockerArgs, "-e", e)
		}
	}

	if stdin != "" {
		dockerArgs = append(dockerArgs, "-i")
	}

	dockerArgs = append(dockerArgs, "--memory")
	if memLimit > 0 {
		dockerArgs = append(dockerArgs, fmt.Sprintf("%dm", memLimit))
	} else {
		dockerArgs = append(dockerArgs, "256m")
	}

	dockerArgs = append(dockerArgs, "--cpus", "1")
	dockerArgs = append(dockerArgs, "--read-only")
	dockerArgs = append(dockerArgs, "--tmpfs", "/tmp:rw,exec,nosuid,size=256m")
	dockerArgs = append(dockerArgs, "-v", workDir+":/code:ro")
	dockerArgs = append(dockerArgs, "-w", "/code")
	dockerArgs = append(dockerArgs, "--security-opt", "no-new-privileges:true")
	dockerArgs = append(dockerArgs, "--cap-drop", "ALL")
	dockerArgs = append(dockerArgs, image)
	dockerArgs = append(dockerArgs, cmdArgs...)

	timeout := 30
	if timeLimit > timeout {
		timeout = timeLimit
	}
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(execCtx, "docker", dockerArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start).Milliseconds()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if execCtx.Err() == context.DeadlineExceeded {
			return &models.ExecuteResponse{
				Stderr:   "Execution timed out",
				ExitCode: -1,
				TimeMs:   elapsed,
				Error:    "timeout",
			}, nil
		} else {
			return &models.ExecuteResponse{
				Error: fmt.Sprintf("docker execution failed: %v", err),
			}, nil
		}
	}

	return &models.ExecuteResponse{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		TimeMs:   elapsed,
	}, nil
}

func (s *SandboxService) generateTestHarness(language, testCases string) (string, error) {
	switch strings.ToLower(language) {
	case "python", "py":
		return s.pyTestHarness(testCases), nil
	case "javascript", "js", "node":
		return s.jsTestHarness(testCases), nil
	case "typescript", "ts":
		return s.jsTestHarness(testCases), nil
	case "go", "golang":
		return s.goTestHarness(testCases), nil
	default:
		return "", fmt.Errorf("function test not supported for language: %s", language)
	}
}

func (s *SandboxService) pyTestHarness(testCases string) string {
	return fmt.Sprintf(`import sys
import json
from solution import *

def run_tests():
    tests = json.loads(%s)
    for i, test in enumerate(tests):
        try:
            fn = globals().get(test["func"])
            if fn is None:
                print(f"Test {i}: FAIL - function %s not found" %% test["func"])
                sys.exit(1)
            result = fn(*test["args"])
            expected = test["expected"]
            assert result == expected, f"Test {i}: got {result}, expected {expected}"
            print(f"Test {i}: PASS")
        except Exception as e:
            print(f"Test {i}: FAIL - {e}")
            sys.exit(1)
    print("ALL TESTS PASSED")

if __name__ == "__main__":
    run_tests()
`, strconv.Quote(testCases))
}

func (s *SandboxService) jsTestHarness(testCases string) string {
	return fmt.Sprintf(`require('./solution.js');

const tests = %s;

function run() {
    tests.forEach((test, i) => {
        try {
            const fn = globalThis[test.func];
            if (typeof fn !== "function") {
                console.log("Test " + i + ": FAIL - function " + test.func + " not found");
                process.exit(1);
            }
            const result = fn(...test.args);
            const expected = test.expected;
            const match = Array.isArray(expected)
                ? JSON.stringify(result) === JSON.stringify(expected)
                : result === expected;
            if (!match) {
                console.log("Test " + i + ": FAIL - got " + JSON.stringify(result) + ", expected " + JSON.stringify(expected));
                process.exit(1);
            }
            console.log("Test " + i + ": PASS");
        } catch (e) {
            console.log("Test " + i + ": FAIL - " + e.message);
            process.exit(1);
        }
    });
    console.log("ALL TESTS PASSED");
}
run();
`, testCases)
}

func (s *SandboxService) goTestHarness(testCases string) string {
	// Parse test cases to know which functions to generate switch entries for
	type tc struct {
		Func string `json:"func"`
	}
	var testList []tc
	json.Unmarshal([]byte(testCases), &testList)

	seen := map[string]bool{}
	var switchCases strings.Builder
	for _, t := range testList {
		if seen[t.Func] {
			continue
		}
		seen[t.Func] = true
		switchCases.WriteString(fmt.Sprintf("    case %s:\n        fn = reflect.ValueOf(%s)\n", strconv.Quote(t.Func), t.Func))
	}

	return fmt.Sprintf(`package main

import (
    "encoding/json"
    "fmt"
    "os"
    "reflect"
)

type TestCase struct {
    Func     string        `+"`json:\"func\"`"+`
    Args     []interface{} `+"`json:\"args\"`"+`
    Expected interface{}   `+"`json:\"expected\"`"+`
}

func callFunc(name string, args []interface{}) (result interface{}, callErr error) {
    defer func() {
        if r := recover(); r != nil {
            callErr = fmt.Errorf("panic: %%v", r)
            result = nil
        }
    }()
    var fn reflect.Value
    switch name {
%s    default:
        return nil, fmt.Errorf("unknown function: %%s", name)
    }
    if fn.Type().NumIn() != len(args) {
        return nil, fmt.Errorf("expected %%d args, got %%d", fn.Type().NumIn(), len(args))
    }
    in := make([]reflect.Value, len(args))
    for i, a := range args {
        fnType := fn.Type().In(i)
        if fnType.Kind() == reflect.Slice {
            src := reflect.ValueOf(a)
            if src.IsValid() && src.Kind() == reflect.Slice {
                dst := reflect.MakeSlice(fnType, src.Len(), src.Len())
                for j := 0; j < src.Len(); j++ {
                    elem := src.Index(j).Interface()
                    if n, ok := elem.(float64); ok && fnType.Elem().Kind() == reflect.Int {
                        dst.Index(j).Set(reflect.ValueOf(int(n)))
                    } else {
                        dst.Index(j).Set(reflect.ValueOf(elem))
                    }
                }
                in[i] = dst
                continue
            }
        }
        if n, ok := a.(float64); ok {
            in[i] = reflect.ValueOf(int(n))
        } else {
            in[i] = reflect.ValueOf(a)
        }
    }
    results := fn.Call(in)
    if len(results) == 0 {
        return nil, nil
    }
    return results[0].Interface(), nil
}

func normalize(v interface{}) interface{} {
    if v == nil {
        return nil
    }
    rv := reflect.ValueOf(v)
    switch rv.Kind() {
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        return float64(rv.Int())
    case reflect.Float32, reflect.Float64:
        return rv.Float()
    case reflect.Slice:
        r := make([]interface{}, rv.Len())
        for i := 0; i < rv.Len(); i++ {
            r[i] = normalize(rv.Index(i).Interface())
        }
        return r
    default:
        return v
    }
}

func main() {
    var tests []TestCase
    if err := json.Unmarshal([]byte(%s), &tests); err != nil {
        fmt.Println("FAIL - invalid tests:", err)
        os.Exit(1)
    }
    for i, test := range tests {
        result, err := callFunc(test.Func, test.Args)
        if err != nil {
            fmt.Printf("Test %%d: FAIL - %%s\\n", i, err)
            os.Exit(1)
        }
        if reflect.DeepEqual(normalize(result), normalize(test.Expected)) {
            fmt.Printf("Test %%d: PASS\\n", i)
        } else {
            fmt.Printf("Test %%d: FAIL - got %%v, expected %%v\\n", i, result, test.Expected)
            os.Exit(1)
        }
    }
    fmt.Println("ALL TESTS PASSED")
}
`, switchCases.String(), strconv.Quote(testCases))
}

func (s *SandboxService) buildFrameworkTestCmd(language string) []string {
	switch strings.ToLower(language) {
	case "python", "py":
		return []string{"sh", "-c", "pip install -r requirements.txt -q 2>/dev/null; python -m pytest -v"}
	case "javascript", "js", "node":
		return []string{"sh", "-c", "npm install --silent 2>/dev/null; npm test 2>/dev/null"}
	case "typescript", "ts":
		return []string{"sh", "-c", "npm install --silent 2>/dev/null; npm test 2>/dev/null"}
	case "go", "golang":
		return []string{"sh", "-c", "go test -v ./..."}
	case "java":
		return []string{"sh", "-c", "mvn test -q 2>/dev/null"}
	default:
		return []string{"npm test"}
	}
}

func (s *SandboxService) copyTemplate(workDir, templateID, language string) error {
	templateDir := filepath.Join("templates", language, templateID)
	if templateID == "" {
		templateDir = filepath.Join("templates", language, "default")
	}
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		_ = os.MkdirAll(workDir, 0755)
		return nil
	}
	return filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(templateDir, path)
		if relPath == "." {
			return nil
		}
		target := filepath.Join(workDir, relPath)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, rErr := os.ReadFile(path)
		if rErr != nil {
			return rErr
		}
		return os.WriteFile(target, data, 0644)
	})
}

func replaceFile(template []string, file string) []string {
	out := make([]string, len(template))
	for i, s := range template {
		out[i] = strings.ReplaceAll(s, "{file}", file)
	}
	return out
}
