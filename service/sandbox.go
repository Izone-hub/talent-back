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

	HarnessExt string            // extension for the generated function-mode harness (defaults to Ext)
	FnRunCmd   []string          // optional override command for function-mode execution ({file} = harness path)
	ExtraFiles map[string]string // auxiliary files written into the work dir (e.g. runners)
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
	case "dart":
		return langConfig{
			Image:    "dart:stable",
			Ext:      ".dart",
			Filename: "main.dart",
			RunCmd:   []string{"dart", "run", "{file}"},
			TestCmd:  []string{"sh", "-c", "cd /code && dart test"},
			Env:      []string{"HOME=/tmp", "PUB_CACHE=/tmp/pubcache"},
		}, nil
	case "sql", "sqlite":
		return langConfig{
			Image:      "python:3.12-slim",
			Ext:        ".sql",
			RunCmd:     []string{"python", "/code/__sql_runner__.py", "{file}"},
			HarnessExt: ".py",
			FnRunCmd:   []string{"python", "/code/test_runner.py"},
			ExtraFiles: map[string]string{"__sql_runner__.py": sqlRunnerScript},
		}, nil
	case "react", "vue", "svelte":
		return langConfig{
			Image:  "sandbox-node:22",
			Ext:    ".jsx",
			RunCmd: []string{"node", "{file}"},
		}, nil
	case "express":
		return langConfig{
			Image:  "sandbox-node:22",
			Ext:    ".js",
			RunCmd: []string{"node", "{file}"},
		}, nil
	case "nextjs", "next.js", "next":
		return langConfig{
			Image:  "sandbox-node:22",
			Ext:    ".tsx",
			RunCmd: []string{"node", "{file}"},
		}, nil
	case "flutter":
		return langConfig{
			Image:  "sandbox-flutter:3.27",
			Ext:    ".dart",
			RunCmd: []string{"flutter", "test", "{file}"},
			Env:    []string{"HOME=/tmp"},
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

func (s *SandboxService) ParseCode(ctx context.Context, req models.ParseRequest) (*models.ParseResponse, error) {
	switch strings.ToLower(req.Language) {
	case "python", "py":
		return s.parsePython(ctx, req.Code)
	case "javascript", "js", "node":
		return s.parseJavaScript(ctx, req.Code)
	default:
		return &models.ParseResponse{Error: fmt.Sprintf("parsing not supported for language: %s", req.Language)}, nil
	}
}

func (s *SandboxService) parsePython(ctx context.Context, code string) (*models.ParseResponse, error) {
	parserScript := `import ast
import json
import sys

code = sys.stdin.read()
tree = ast.parse(code)

functions = []
for node in ast.walk(tree):
    if isinstance(node, ast.FunctionDef):
        args = [arg.arg for arg in node.args.args]
        functions.append({
            "name": node.name,
            "args": args,
            "startLine": node.lineno,
            "endLine": node.end_lineno or node.lineno
        })

print(json.dumps({"functions": functions}))
`

	workDir, err := os.MkdirTemp("", "sandbox-parse-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create work dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	parserFile := filepath.Join(workDir, "parser.py")
	if err := os.WriteFile(parserFile, []byte(parserScript), 0644); err != nil {
		return nil, fmt.Errorf("failed to write parser: %w", err)
	}

	dockerArgs := []string{
		"run", "--rm", "--network", "none",
		"--memory", "128m", "--cpus", "0.5",
		"--read-only", "--tmpfs", "/tmp:rw,exec,nosuid,size=64m",
		"-v", workDir+":/code:ro", "-w", "/code",
		"--security-opt", "no-new-privileges:true",
		"--cap-drop", "ALL",
		"python:3.12-slim", "python", "/code/parser.py",
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(code)

	if err := cmd.Run(); err != nil {
		return &models.ParseResponse{Error: stderr.String()}, nil
	}

	var result models.ParseResponse
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return &models.ParseResponse{Error: "failed to parse output: " + err.Error()}, nil
	}
	return &result, nil
}

func (s *SandboxService) parseJavaScript(ctx context.Context, code string) (*models.ParseResponse, error) {
	parserScript := `const acorn = require('acorn');
const code = require('fs').readFileSync('/code/input.js', 'utf8');

try {
    const ast = acorn.parse(code, { ecmaVersion: 2022, sourceType: 'module' });
    const functions = [];
    
    for (const node of ast.body) {
        if (node.type === 'FunctionDeclaration') {
            functions.push({
                name: node.id.name,
                args: node.params.map(p => p.name),
                startLine: node.loc.start.line,
                endLine: node.loc.end.line
            });
        }
    }
    
    console.log(JSON.stringify({ functions }));
} catch (e) {
    console.log(JSON.stringify({ functions: [], error: e.message }));
}
`

	workDir, err := os.MkdirTemp("", "sandbox-parse-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create work dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	parserFile := filepath.Join(workDir, "parser.js")
	if err := os.WriteFile(parserFile, []byte(parserScript), 0644); err != nil {
		return nil, fmt.Errorf("failed to write parser: %w", err)
	}

	inputFile := filepath.Join(workDir, "input.js")
	if err := os.WriteFile(inputFile, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write input: %w", err)
	}

	packageJSON := filepath.Join(workDir, "package.json")
	if err := os.WriteFile(packageJSON, []byte(`{"dependencies":{"acorn":"^8.11.0"}}`), 0644); err != nil {
		return nil, fmt.Errorf("failed to write package.json: %w", err)
	}

	dockerArgs := []string{
		"run", "--rm", "--network", "none",
		"--memory", "128m", "--cpus", "0.5",
		"--read-only", "--tmpfs", "/tmp:rw,exec,nosuid,size=64m",
		"-v", workDir+":/code", "-w", "/code",
		"--security-opt", "no-new-privileges:true",
		"--cap-drop", "ALL",
		"node:22-slim", "sh", "-c", "npm install --silent 2>/dev/null && node /code/parser.js",
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return &models.ParseResponse{Error: stderr.String()}, nil
	}

	var result models.ParseResponse
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return &models.ParseResponse{Error: "failed to parse output: " + err.Error()}, nil
	}
	return &result, nil
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

	for name, content := range cfg.ExtraFiles {
		if err := os.WriteFile(filepath.Join(workDir, name), []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", name, err)
		}
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
	harnessExt := cfg.HarnessExt
	if harnessExt == "" {
		harnessExt = cfg.Ext
	}
	testFile := filepath.Join(workDir, "test_runner"+harnessExt)
	if err := os.WriteFile(testFile, []byte(testHarness), 0644); err != nil {
		return nil, fmt.Errorf("failed to write test harness: %w", err)
	}

	for name, content := range cfg.ExtraFiles {
		if err := os.WriteFile(filepath.Join(workDir, name), []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", name, err)
		}
	}

	// Additional request files (e.g. SQL questions ship schema.sql /
	// seed.sql so the harness can rebuild the imported database).
	for path, content := range req.Files {
		full := filepath.Join(workDir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			return nil, fmt.Errorf("failed to create dir for %s: %w", path, err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", path, err)
		}
	}

	// Always write test cases to a JSON file so harnesses can read from disk
	// instead of embedding JSON in source code (avoids escaping issues)
	testsJSONFile := filepath.Join(workDir, "tests.json")
	_ = os.WriteFile(testsJSONFile, []byte(req.Stdin), 0644)

	runTemplate := cfg.RunCmd
	if len(cfg.FnRunCmd) > 0 {
		runTemplate = cfg.FnRunCmd
	}
	cmdArgs := replaceFile(runTemplate, "/code/test_runner"+harnessExt)
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
	case "dart":
		return s.dartTestHarness(testCases), nil
	case "sql", "sqlite":
		return s.sqlTestHarness(testCases), nil
	default:
		return "", fmt.Errorf("function test not supported for language: %s", language)
	}
}

func (s *SandboxService) dartTestHarness(testCases string) string {
	return `import 'dart:convert';
import 'dart:io';
import 'dart:mirrors';

import 'solution.dart';

const _invalidStrings = {'', '-', '--', '.', 'null', 'none', 'undefined', 'nan', 'inf', 'infinity'};

bool _invalid(String v) => _invalidStrings.contains(v.trim().toLowerCase());

bool deepEq(dynamic a, dynamic b) {
  if (identical(a, b)) return true;
  if (a is num && b is num) {
    if (a is double && b is int) return a == b.toDouble();
    if (a is int && b is double) return a.toDouble() == b;
    return a == b;
  }
  if (a is List && b is List) {
    if (a.length != b.length) return false;
    for (var i = 0; i < a.length; i++) {
      if (!deepEq(a[i], b[i])) return false;
    }
    return true;
  }
  if (a is Map && b is Map) {
    if (a.length != b.length) return false;
    for (final k in a.keys) {
      if (!b.containsKey(k) || !deepEq(a[k], b[k])) return false;
    }
    return true;
  }
  return a == b;
}

dynamic norm(dynamic v) {
  if (v is double && !v.isInfinite && v == v.truncateToDouble()) return v.toInt();
  if (v is List) return v.map(norm).toList();
  if (v is Map) return v.map((k, val) => MapEntry(k.toString(), norm(val)));
  return v;
}

TypeMirror _target(Type type) => reflectType(type);

bool _isA(TypeMirror t, Type type) {
  final target = _target(type);
  return t == target || t.isSubtypeOf(target);
}

bool _isList(TypeMirror t) => _isA(t, List);
bool _isMap(TypeMirror t) => _isA(t, Map);

// _convert turns a decoded JSON value into an instance matching the
// declared parameter type of the user's function.
dynamic _convert(dynamic v, TypeMirror t) {
  if (v == null) return null;
  if (_isA(t, int)) {
    if (v is num) return v.toInt();
    if (v is String) return int.tryParse(v.trim());
    return null;
  }
  if (_isA(t, double)) {
    if (v is num) return v.toDouble();
    if (v is String) return double.tryParse(v.trim());
    return null;
  }
  if (_isA(t, num)) {
    if (v is num) return v;
    return num.tryParse(v.toString());
  }
  if (_isA(t, String)) return v.toString();
  if (_isA(t, bool)) {
    if (v is bool) return v;
    return v.toString() == 'true';
  }
  if (_isList(t)) {
    TypeMirror elem = reflectType(dynamic);
    if (t is ClassMirror && t.typeArguments.isNotEmpty) elem = t.typeArguments[0];
    final src = (v is List ? v : [v]).map((e) => _convert(e, elem)).toList();
    // Dart generics are reified: a List<dynamic> cannot flow into a
    // List<int> parameter, so rebuild the collection with its exact type.
    if (t is ClassMirror && !identical(elem, reflectType(dynamic))) {
      try {
        return t.newInstance(#from, [src]).reflectee;
      } catch (_) {}
    }
    return src;
  }
  if (_isMap(t)) {
    TypeMirror val = reflectType(dynamic);
    if (t is ClassMirror && t.typeArguments.length > 1) val = t.typeArguments[1];
    final out = <dynamic, dynamic>{};
    if (v is Map) {
      v.forEach((k, e) {
        out[k.toString()] = _convert(e, val);
      });
    }
    if (t is ClassMirror && !identical(val, reflectType(dynamic))) {
      try {
        return t.newInstance(#from, [out]).reflectee;
      } catch (_) {}
    }
    return out;
  }
  return v;
}

// _auto mirrors the JS/Python harnesses: numeric strings become numbers,
// stringified JSON ("[1,2,3]") becomes real structures.
dynamic _auto(dynamic v) {
  if (v is String) {
    final s = v.trim();
    if (_invalid(s)) return null;
    if ((s.startsWith('[') && s.endsWith(']')) || (s.startsWith('{') && s.endsWith('}'))) {
      try {
        return _auto(jsonDecode(s));
      } catch (_) {}
    }
    return num.tryParse(s) ?? v;
  }
  if (v is List) return v.map(_auto).toList();
  if (v is Map) return v.map((k, e) => MapEntry(k.toString(), _auto(e)));
  return v;
}

void main() {
  LibraryMirror? sol;
  currentMirrorSystem().libraries.forEach((uri, lib) {
    if (sol == null && uri.toString().endsWith('/solution.dart')) {
      sol = lib;
    }
  });
  sol ??= currentMirrorSystem().isolate.rootLibrary;
  if (sol == null) {
    print('FAIL - cannot locate solution library');
    exit(1);
  }

  final tests = jsonDecode(File('/code/tests.json').readAsStringSync()) as List;
  for (var i = 0; i < tests.length; i++) {
    final t = tests[i] as Map;
    final name = t['func'] as String? ?? '';
    final sym = Symbol(name);
    final decl = sol!.declarations[sym];
    if (decl is! MethodMirror) {
      print('Test $i: FAIL - function $name not found');
      exit(1);
    }
    try {
      final rawArgs = t['args'];
      final provided = rawArgs is List ? rawArgs : (rawArgs == null ? [] : [rawArgs]);

      var skip = false;
      for (final a in provided) {
        if (a is String && _invalid(a)) {
          print('Test $i: SKIP - invalid input $a');
          skip = true;
          break;
        }
      }
      if (skip) continue;

      final params = decl.parameters;
      final args = <Object?>[];
      final n = provided.length < params.length ? provided.length : params.length;
      for (var j = 0; j < n; j++) {
        args.add(_convert(provided[j], params[j].type));
      }

      final result = sol!.invoke(sym, args).reflectee;
      final expected = _auto(t['expected']);
      if (deepEq(norm(result), norm(expected))) {
        print('Test $i: PASS');
      } else {
        print('Test $i: FAIL - got $result, expected ${jsonEncode(expected)}');
        exit(1);
      }
    } catch (e) {
      print('Test $i: FAIL - $e');
      exit(1);
    }
  }
  print('ALL TESTS PASSED');
}
`
}

// sqlRunnerScript executes a .sql file against an in-memory SQLite database.
// If schema.sql / seed.sql exist next to it they are applied first.
const sqlRunnerScript = `import os
import sqlite3
import sys

def apply_file(con, path):
    with open(path) as f:
        con.executescript(f.read())

def run_statement(cur, stmt):
    cur.execute(stmt)
    rows = cur.fetchall()
    if rows:
        if cur.description:
            print("|".join(d[0] or "" for d in cur.description))
        for r in rows:
            print("|".join("NULL" if v is None else str(v) for v in r))
        print()

def main():
    target = sys.argv[1]
    con = sqlite3.connect(":memory:")
    con.isolation_level = None
    cur = con.cursor()
    for aux in ("/code/schema.sql", "/code/seed.sql"):
        if os.path.exists(aux):
            apply_file(con, aux)

    with open(target) as f:
        sql = f.read()

    stmt = ""
    for line in sql.splitlines():
        stmt += line + "\n"
        if sqlite3.complete_statement(stmt):
            s = stmt.strip()
            stmt = ""
            if not s:
                continue
            try:
                run_statement(cur, s)
            except Exception as e:
                print(f"ERROR: {e}", file=sys.stderr)
                sys.exit(1)
    if stmt.strip():
        try:
            run_statement(cur, stmt.strip())
        except Exception as e:
            print(f"ERROR: {e}", file=sys.stderr)
            sys.exit(1)

main()
`

func (s *SandboxService) sqlTestHarness(testCases string) string {
	return `import json
import os
import sqlite3
import sys

SOLUTION_PATH = "/code/solution.sql"


def fresh_db(setup_sql=None):
    """Rebuild the imported database: schema + seed (+ per-test setup)."""
    con = sqlite3.connect(":memory:")
    for aux in ("/code/schema.sql", "/code/seed.sql"):
        if os.path.exists(aux):
            with open(aux) as f:
                con.executescript(f.read())
    if setup_sql:
        con.executescript(setup_sql)
    return con


def norm(v):
    if isinstance(v, bool):
        return int(v)
    if isinstance(v, float) and v.is_integer() and abs(v) < 1e15:
        return int(v)
    return v


def norm_rows(rows):
    return [[norm(c) for c in row] for row in rows]


def wrap(expected):
    # Accept scalars, single rows, or full row sets.
    if expected is None:
        return None
    if not isinstance(expected, list):
        return [[norm(expected)]]
    if expected and not any(isinstance(x, (list, tuple)) for x in expected):
        return [[norm(x) for x in expected]]
    return norm_rows(expected)


def compare(actual, expected, ordered=True):
    exp = wrap(expected)
    act = norm_rows(actual)
    if exp is None:
        return True
    if not ordered:
        def key(rows):
            return sorted(json.dumps(r, sort_keys=True, default=str) for r in rows)
        return key(act) == key(exp)
    return act == exp


def main():
    with open("/code/tests.json") as f:
        tests = json.load(f)

    solution = ""
    if os.path.exists(SOLUTION_PATH):
        with open(SOLUTION_PATH) as f:
            solution = f.read()

    has_solution = bool(solution.strip())

    for i, t in enumerate(tests):
        label = t.get("name") or ("test %d" % i)
        try:
            con = fresh_db(t.get("setup"))
            cur = con.cursor()
            verify = t.get("verify")
            query = t.get("query")
            if verify:
                # Write task (INSERT/UPDATE/DELETE/DDL): apply the student's
                # statements first, then run the verification query against
                # the resulting database state.
                if has_solution:
                    con.executescript(solution)
                cur.execute(verify)
            elif query is not None or t.get("expected_rows", t.get("expected")) is not None:
                # Read task: run the given query on a fresh imported DB.
                stmt = query if query else solution
                if not (stmt or "").strip():
                    print("Test %d (%s): FAIL - no query to execute" % (i, label))
                    sys.exit(1)
                cur.execute(stmt)
            elif has_solution:
                # No explicit query: treat the student's SQL itself as the
                # statement whose result set we compare.
                cur.execute(solution)
            else:
                print("Test %d (%s): FAIL - nothing to execute" % (i, label))
                sys.exit(1)
            actual = cur.fetchall()
            con.close()
        except Exception as e:
            print("Test %d (%s): FAIL - %s" % (i, label, e))
            sys.exit(1)

        expected = t.get("expected_rows", t.get("expected"))
        ordered = t.get("ordered", True)
        if compare(actual, expected, ordered):
            print("Test %d (%s): PASS" % (i, label))
        else:
            print("Test %d (%s): FAIL - got %s, expected %s"
                  % (i, label, json.dumps(norm_rows(actual)), json.dumps(wrap(expected))))
            sys.exit(1)
    print("ALL TESTS PASSED")


main()
`
}

func (s *SandboxService) pyTestHarness(testCases string) string {
	return fmt.Sprintf(`import sys
import json
import re
import inspect
from solution import *

INVALID_STRINGS = {'', '-', '--', '.', 'null', 'None', 'undefined', 'nan', 'inf', 'none', 'nan', 'infinity'}

def is_invalid_string(val):
    """Check if a string value is invalid / non-numeric placeholder."""
    if not isinstance(val, str):
        return False
    return val.strip().lower() in INVALID_STRINGS

def looks_like_json(val):
    """Check if a string looks like a JSON array or object."""
    if not isinstance(val, str):
        return False
    t = val.strip()
    return (t.startswith('[') and t.endswith(']')) or (t.startswith('{') and t.endswith('}'))

def is_numeric_string(val):
    """Check if a string is a valid number (int or float)."""
    if not isinstance(val, str):
        return False
    t = val.strip()
    if not t or t in INVALID_STRINGS:
        return False
    # Match optional sign, digits, optional decimal, optional exponent
    return bool(re.match(r'^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$', t))

def auto_convert(val):
    if isinstance(val, str):
        val = val.strip()
        if is_invalid_string(val):
            return None
        # Try parsing JSON (handles "[1,2,3]" -> [1,2,3] and "{}" -> {})
        if looks_like_json(val):
            try:
                parsed = json.loads(val)
                if isinstance(parsed, list):
                    return [auto_convert(x) for x in parsed]
                return parsed
            except (json.JSONDecodeError, ValueError):
                # Try normalizing single quotes to double quotes
                # (handles Python-style lists like ['a','b','c'])
                try:
                    normalized = val.replace("'", '"')
                    parsed = json.loads(normalized)
                    if isinstance(parsed, list):
                        return [auto_convert(x) for x in parsed]
                    return parsed
                except (json.JSONDecodeError, ValueError):
                    pass
        # Try numeric conversion
        if is_numeric_string(val):
            try:
                return int(val)
            except ValueError:
                try:
                    return float(val)
                except ValueError:
                    pass
        return val
    if isinstance(val, list):
        return [auto_convert(x) for x in val]
    if isinstance(val, dict):
        return {k: auto_convert(v) for k, v in val.items()}
    return val

def run_tests():
    tests = json.loads(%s)
    skipped = 0
    for i, test in enumerate(tests):
        try:
            fn = globals().get(test["func"])
            if fn is None:
                print("Test %%d: FAIL - function %%s not found" %% (i, test["func"]))
                sys.exit(1)
            sig = inspect.signature(fn)
            param_count = len(sig.parameters)
            raw_args = test["args"]
            # Auto-convert the args (handles stringified lists, numbers, etc.)
            if isinstance(raw_args, list):
                # Check for invalid args BEFORE calling the function
                for a in raw_args:
                    if is_invalid_string(a):
                        print("Test %%d: SKIP - invalid input %%s" %% (i, repr(a)))
                        skipped += 1
                        break
                else:
                    all_args = [auto_convert(a) for a in raw_args]
                    args = all_args[:param_count]
                    result = fn(*args)
                    expected = auto_convert(test["expected"])
                    strict = test.get("strict", False)
                    if strict:
                        match = result == expected
                    else:
                        # Relaxed: None and False/0/empty are not silently equivalent
                        match = result == expected
                    assert match, "Test %%d: got %%s, expected %%s" %% (i, result, expected)
                    print("Test %%d: PASS" %% i)
            else:
                all_args = [auto_convert(raw_args)]
                args = all_args[:param_count]
                result = fn(*args)
                expected = auto_convert(test["expected"])
                strict = test.get("strict", False)
                if strict:
                    match = result == expected
                else:
                    match = result == expected
                assert match, "Test %%d: got %%s, expected %%s" %% (i, result, expected)
                print("Test %%d: PASS" %% i)
        except Exception as e:
            print("Test %%d: FAIL - %%s" %% (i, e))
            sys.exit(1)
    print("ALL TESTS PASSED")

if __name__ == "__main__":
    run_tests()
`, strconv.Quote(testCases))
}

func (s *SandboxService) jsTestHarness(testCases string) string {
	return fmt.Sprintf(`const fs = require('fs');
const vm = require('vm');
const code = fs.readFileSync('/code/solution.js', 'utf8');
vm.runInThisContext(code, { filename: 'solution.js' });

const tests = %s;

const INVALID_STRINGS = new Set(['', '-', '--', '.', 'null', 'undefined', 'nan', 'inf', 'none', 'infinity']);

function isInvalidString(val) {
    if (typeof val !== 'string') return false;
    return INVALID_STRINGS.has(val.trim().toLowerCase());
}

function looksLikeJson(val) {
    if (typeof val !== 'string') return false;
    const t = val.trim();
    return (t.startsWith('[') && t.endsWith(']')) || (t.startsWith('{') && t.endsWith('}'));
}

function isNumericString(val) {
    if (typeof val !== 'string') return false;
    const t = val.trim();
    if (!t || INVALID_STRINGS.has(t.toLowerCase())) return false;
    return /^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$/.test(t);
}

function autoConvert(val) {
    if (typeof val === 'string') {
        const trimmed = val.trim();
        if (isInvalidString(val)) return null;
        // Try parsing JSON (handles "[1,2,3]" -> [1,2,3] and "{}" -> {})
        if (looksLikeJson(val)) {
            try {
                const parsed = JSON.parse(trimmed);
                if (Array.isArray(parsed)) return parsed.map(autoConvert);
                return parsed;
            } catch (e) {
                // Try normalizing single quotes to double quotes
                // (handles Python-style lists like ['a','b','c'])
                try {
                    const normalized = trimmed.replace(/'/g, '"');
                    const parsed = JSON.parse(normalized);
                    if (Array.isArray(parsed)) return parsed.map(autoConvert);
                    return parsed;
                } catch (e2) {
                    // Not valid even after normalization
                }
            }
        }
        // Try numeric conversion
        if (isNumericString(val)) {
            const num = Number(trimmed);
            if (!isNaN(num)) return num;
        }
        return val;
    }
    if (Array.isArray(val)) return val.map(autoConvert);
    if (val && typeof val === 'object') {
        const out = {};
        for (const [k, v] of Object.entries(val)) out[k] = autoConvert(v);
        return out;
    }
    return val;
}

function getParamCount(fn) {
    return fn.length;
}

function run() {
    tests.forEach((test, i) => {
        try {
            const fn = globalThis[test.func];
            if (typeof fn !== "function") {
                console.log("Test " + i + ": FAIL - function " + test.func + " not found");
                process.exit(1);
            }
            // Auto-convert the args (handles stringified lists, numbers, etc.)
            const rawArgs = test.args;
            if (Array.isArray(rawArgs)) {
                // Check for invalid args BEFORE calling the function
                for (const a of rawArgs) {
                    if (isInvalidString(a)) {
                        console.log("Test " + i + ": SKIP - invalid input " + JSON.stringify(a));
                        return;
                    }
                }
            }
            const allArgs = (Array.isArray(rawArgs) ? rawArgs : [rawArgs]).map(autoConvert);
            const paramCount = getParamCount(fn);
            const args = allArgs.slice(0, paramCount);
            const result = fn(...args);
            const expected = autoConvert(test.expected);
            const strict = test.strict === true;
            let match;
            if (Array.isArray(expected)) {
                match = JSON.stringify(result) === JSON.stringify(expected);
            } else if (strict) {
                match = result === expected;
            } else {
                // Relaxed mode: treat null and undefined as equivalent
                match = result == expected || (result == null && expected == null);
            }
            if (!match) {
                console.log("Test " + i + ": FAIL - got " + JSON.stringify(result) + ", expected " + JSON.stringify(expected) + (strict ? " (strict)" : " (relaxed)"));
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
		switchCases.WriteString(fmt.Sprintf("    case \"%s\":\n        fn = reflect.ValueOf(%s)\n", t.Func, t.Func))
	}

	// Build Go source using string replacement instead of fmt.Sprintf
	// to avoid all the %% escaping nightmares
	goSrc := `package main

import (
    "encoding/json"
    "fmt"
    "os"
    "reflect"
    "strconv"
    "strings"
)

type TestCase struct {
    Func     string          __TAG_FUNC__
    Args     json.RawMessage __TAG_ARGS__
    Expected json.RawMessage __TAG_EXPECTED__
}

func parseArgs(raw json.RawMessage) (interface{}, error) {
    if len(raw) == 0 {
        return nil, nil
    }
    // First try: parse as-is (array, object, number, bool, null)
    var val interface{}
    if err := json.Unmarshal(raw, &val); err == nil {
        // If the result is a string that looks like JSON (e.g. "[1,2,3]" or
        // "42"), try to re-parse the string content as JSON.
        if s, ok := val.(string); ok {
            return parseStringifiedValue(s)
        }
        return val, nil
    }
    return nil, fmt.Errorf("cannot parse args: %s", string(raw))
}

// parseStringifiedValue handles the case where a JSON value is a string
// that contains a JSON-encoded value, e.g. "[1,2,3]" or "42".
func parseStringifiedValue(s string) (interface{}, error) {
    s = strings.TrimSpace(s)
    // Try parsing the string content as JSON (list, object, number)
    var inner interface{}
    if err := json.Unmarshal([]byte(s), &inner); err == nil {
        return inner, nil
    }
    // Try numeric
    if n, err := strconv.ParseFloat(s, 64); err == nil {
        return n, nil
    }
    return s, nil
}

func autoConvertNums(v interface{}) interface{} {
    switch val := v.(type) {
    case float64:
        if val == float64(int(val)) {
            return int(val)
        }
        return val
    case []interface{}:
        out := make([]interface{}, len(val))
        for i, x := range val {
            out[i] = autoConvertNums(x)
        }
        return out
    case map[string]interface{}:
        out := make(map[string]interface{}, len(val))
        for k, x := range val {
            out[k] = autoConvertNums(x)
        }
        return out
    case string:
        s := strings.TrimSpace(val)
        if s == "" || s == "-" || s == "null" || s == "undefined" || s == "nan" || s == "inf" {
            return nil
        }
        // Try numeric first
        if n, err := strconv.ParseFloat(s, 64); err == nil {
            return autoConvertNums(n)
        }
        // Try parsing as JSON (handles stringified lists/objects like "[1,2,3]")
        if (strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")) ||
            (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) {
            var inner interface{}
            if err := json.Unmarshal([]byte(s), &inner); err == nil {
                return autoConvertNums(inner)
            }
        }
        return val
    default:
        return v
    }
}

func callFunc(name string, args []interface{}) (result interface{}, callErr error) {
    defer func() {
        if r := recover(); r != nil {
            callErr = fmt.Errorf("panic: %v", r)
            result = nil
        }
    }()
    var fn reflect.Value
    switch name {
__SWITCH_CASES__
    default:
        return nil, fmt.Errorf("unknown function: %s", name)
    }

    ft := fn.Type()
    isVariadic := ft.IsVariadic()
    numFixed := ft.NumIn()
    if isVariadic {
        numFixed-- // last param is the variadic one
    }

    // Count how many args we actually have for fixed params + variadic
    if len(args) < numFixed {
        return nil, fmt.Errorf("expected at least %d args, got %d", numFixed, len(args))
    }

    // Build the reflect.Value slice
    in := make([]reflect.Value, 0, len(args))

    // Fixed parameters
    for i := 0; i < numFixed; i++ {
        in = append(in, convertArg(args[i], ft.In(i)))
    }

    // Variadic parameter
    if isVariadic {
        varType := ft.In(numFixed).Elem() // element type of the variadic slice
        variadicArgs := args[numFixed:]
        // If there's exactly one arg and it's a list, unpack it.
        // This handles the case where the employer enters [1,2,3,4] as
        // a single input string — the harness wraps it as [[1,2,3,4]],
        // but a variadic func expects individual args [1,2,3,4].
        if len(variadicArgs) == 1 {
            if list, ok := variadicArgs[0].([]interface{}); ok {
                variadicArgs = list
            }
        }
        for _, a := range variadicArgs {
            in = append(in, convertArg(a, varType))
        }
    } else {
        // Non-variadic: remaining args
        for i := numFixed; i < len(args) && i < ft.NumIn(); i++ {
            in = append(in, convertArg(args[i], ft.In(i)))
        }
    }

    results := fn.Call(in)
    if len(results) == 0 {
        return nil, nil
    }
    return results[0].Interface(), nil
}

// convertArg converts a single argument from interface{} to the target reflect.Type.
func convertArg(a interface{}, targetType reflect.Type) reflect.Value {
    // Handle stringified JSON values
    if s, ok := a.(string); ok {
        s = strings.TrimSpace(s)
        if s == "" || s == "-" || s == "null" || s == "undefined" || s == "nan" || s == "inf" {
            a = nil
        } else if (strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")) ||
            (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) {
            var inner interface{}
            if err := json.Unmarshal([]byte(s), &inner); err == nil {
                a = inner
            }
        } else if n, err := strconv.ParseFloat(s, 64); err == nil {
            a = n
        }
    }

    // nil handling
    if a == nil {
        return reflect.Zero(targetType)
    }

    // If target is a slice type, build the slice from an array/list
    if targetType.Kind() == reflect.Slice {
        src := reflect.ValueOf(a)
        if src.IsValid() && src.Kind() == reflect.Slice {
            elemType := targetType.Elem()
            dst := reflect.MakeSlice(targetType, src.Len(), src.Len())
            for j := 0; j < src.Len(); j++ {
                elem := src.Index(j).Interface()
                dst.Index(j).Set(convertArg(elem, elemType))
            }
            return dst
        }
        // a is not a slice but target expects one — wrap it
        dst := reflect.MakeSlice(targetType, 1, 1)
        dst.Index(0).Set(convertArg(a, targetType.Elem()))
        return dst
    }

    // Numeric conversion
    if n, ok := a.(float64); ok {
        switch targetType.Kind() {
        case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
            return reflect.ValueOf(int(n)).Convert(targetType)
        case reflect.Float32, reflect.Float64:
            return reflect.ValueOf(n).Convert(targetType)
        }
    }
    if n, ok := a.(int); ok {
        switch targetType.Kind() {
        case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
            return reflect.ValueOf(n).Convert(targetType)
        case reflect.Float32, reflect.Float64:
            return reflect.ValueOf(float64(n)).Convert(targetType)
        }
    }

    val := reflect.ValueOf(a)
    if val.Type().ConvertibleTo(targetType) {
        return val.Convert(targetType)
    }
    return val
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
    data, err := os.ReadFile("/code/tests.json")
    if err != nil {
        fmt.Println("FAIL - cannot read tests.json:", err)
        os.Exit(1)
    }
    var tests []TestCase
    if err := json.Unmarshal(data, &tests); err != nil {
        fmt.Println("FAIL - invalid tests:", err)
        os.Exit(1)
    }
    for i, test := range tests {
        // Detect if the original args value was a JSON string (not an array).
        // e.g. "args": "[1,2,3,4]" is a stringified list — the whole thing
        // is one argument. "args": [[1,2,3,4]] is a proper array of arguments.
        var argsWasString bool
        {
            var check interface{}
            if json.Unmarshal(test.Args, &check) == nil {
                _, argsWasString = check.(string)
            }
        }

        rawArgs, err := parseArgs(test.Args)
        if err != nil {
            fmt.Printf("Test %d: SKIP - bad args: %s\n", i, err)
            continue
        }
        argsList, ok := rawArgs.([]interface{})
        if !ok || argsWasString {
            // If args was a JSON string (like "[1,2,3,4]" or "42"),
            // the parsed value is the argument itself, not a list of arguments.
            argsList = []interface{}{rawArgs}
        }
        for j, a := range argsList {
            argsList[j] = autoConvertNums(a)
        }
        rawExpected, err := parseArgs(test.Expected)
        if err != nil {
            rawExpected = nil
        }
        rawExpected = autoConvertNums(rawExpected)
        result, err := callFunc(test.Func, argsList)
        if err != nil {
            fmt.Printf("Test %d: FAIL - %s\n", i, err)
            os.Exit(1)
        }
        if reflect.DeepEqual(normalize(result), normalize(rawExpected)) {
            fmt.Printf("Test %d: PASS\n", i)
        } else {
            fmt.Printf("Test %d: FAIL - got %v, expected %v\n", i, result, rawExpected)
            os.Exit(1)
        }
    }
    fmt.Println("ALL TESTS PASSED")
}
`
	goSrc = strings.Replace(goSrc, "__SWITCH_CASES__", switchCases.String(), 1)
	tagFunc := "`" + `json:"func"` + "`"
	tagArgs := "`" + `json:"args"` + "`"
	tagExpected := "`" + `json:"expected"` + "`"
	goSrc = strings.Replace(goSrc, "__TAG_FUNC__", tagFunc, 1)
	goSrc = strings.Replace(goSrc, "__TAG_ARGS__", tagArgs, 1)
	goSrc = strings.Replace(goSrc, "__TAG_EXPECTED__", tagExpected, 1)
	return goSrc
}

func (s *SandboxService) buildFrameworkTestCmd(language string) []string {
	switch strings.ToLower(language) {
	case "python", "py":
		return []string{"sh", "-c", "pip install -r requirements.txt -q 2>/dev/null; python -m pytest -v"}
	case "javascript", "js", "node":
		return []string{"sh", "-c", "npm install --silent 2>/dev/null; npm test 2>/dev/null"}
	case "typescript", "ts":
		return []string{"sh", "-c", "npm install --silent 2>/dev/null; npm test 2>/dev/null"}
	case "react", "vue", "svelte", "express", "nextjs":
		return []string{"sh", "-c", "npm install --silent 2>/dev/null; npm test 2>/dev/null"}
	case "flutter":
		return []string{"sh", "-c", "flutter pub get --offline 2>/dev/null; flutter test --reporter expanded"}
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
