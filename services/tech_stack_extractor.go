package services

import (
	"encoding/json"
	"strings"
)

// ExtractTechStackFromFile extracts tech stack from dependency files
func ExtractTechStackFromFile(filename, content string) map[string]string {
	techStack := make(map[string]string)

	switch filename {
	case "package.json":
		techStack = extractFromPackageJson(content)
	case "go.mod":
		techStack = extractFromGoMod(content)
	case "requirements.txt":
		techStack = extractFromRequirementsTxt(content)
	case "pom.xml":
		techStack = extractFromPomXml(content)
	case "Gemfile":
		techStack = extractFromGemfile(content)
	case "Cargo.toml":
		techStack = extractFromCargoToml(content)
	case "composer.json":
		techStack = extractFromComposerJson(content)
	}

	return techStack
}

func extractFromPackageJson(content string) map[string]string {
	techStack := make(map[string]string)

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return techStack
	}

	// Detect framework
	if _, ok := pkg.Dependencies["react"]; ok {
		techStack["framework"] = "React"
	} else if _, ok := pkg.Dependencies["vue"]; ok {
		techStack["framework"] = "Vue"
	} else if _, ok := pkg.Dependencies["angular"]; ok {
		techStack["framework"] = "Angular"
	} else if _, ok := pkg.Dependencies["express"]; ok {
		techStack["framework"] = "Express"
	} else if _, ok := pkg.Dependencies["next"]; ok {
		techStack["framework"] = "Next.js"
	}

	// Detect styling
	if _, ok := pkg.Dependencies["tailwindcss"]; ok {
		techStack["styling"] = "TailwindCSS"
	} else if _, ok := pkg.Dependencies["styled-components"]; ok {
		techStack["styling"] = "Styled Components"
	}

	// Detect build tools
	if _, ok := pkg.DevDependencies["vite"]; ok {
		techStack["build_tool"] = "Vite"
	} else if _, ok := pkg.DevDependencies["webpack"]; ok {
		techStack["build_tool"] = "Webpack"
	}

	// Detect TypeScript
	if _, ok := pkg.Dependencies["typescript"]; ok || pkg.DevDependencies["typescript"] != "" {
		techStack["language"] = "TypeScript"
	} else {
		techStack["language"] = "JavaScript"
	}

	return techStack
}

func extractFromGoMod(content string) map[string]string {
	techStack := make(map[string]string)

	techStack["language"] = "Go"

	// Detect framework
	if strings.Contains(content, "gin-gonic/gin") {
		techStack["framework"] = "Gin"
	} else if strings.Contains(content, "gorilla/mux") {
		techStack["framework"] = "Gorilla Mux"
	} else if strings.Contains(content, "echo") {
		techStack["framework"] = "Echo"
	}

	// Detect database
	if strings.Contains(content, "jackc/pgx") || strings.Contains(content, "lib/pq") {
		techStack["database"] = "PostgreSQL"
	} else if strings.Contains(content, "go-sql-driver/mysql") {
		techStack["database"] = "MySQL"
	} else if strings.Contains(content, "mongodb/mongo-go-driver") {
		techStack["database"] = "MongoDB"
	}

	// Detect auth
	if strings.Contains(content, "golang-jwt/jwt") {
		techStack["auth"] = "JWT"
	}

	return techStack
}

func extractFromRequirementsTxt(content string) map[string]string {
	techStack := make(map[string]string)

	techStack["language"] = "Python"

	// Detect framework
	if strings.Contains(content, "Django") {
		techStack["framework"] = "Django"
	} else if strings.Contains(content, "Flask") {
		techStack["framework"] = "Flask"
	} else if strings.Contains(content, "FastAPI") {
		techStack["framework"] = "FastAPI"
	}

	// Detect database
	if strings.Contains(content, "psycopg2") {
		techStack["database"] = "PostgreSQL"
	} else if strings.Contains(content, "mysql") {
		techStack["database"] = "MySQL"
	}

	// Detect task queue
	if strings.Contains(content, "celery") {
		techStack["task_queue"] = "Celery"
	}

	// Detect cache
	if strings.Contains(content, "redis") {
		techStack["cache"] = "Redis"
	}

	return techStack
}

func extractFromPomXml(content string) map[string]string {
	techStack := make(map[string]string)

	techStack["language"] = "Java"

	// Detect framework
	if strings.Contains(content, "spring-boot") {
		techStack["framework"] = "Spring Boot"
	} else if strings.Contains(content, "springframework") {
		techStack["framework"] = "Spring"
	}

	return techStack
}

func extractFromGemfile(content string) map[string]string {
	techStack := make(map[string]string)

	techStack["language"] = "Ruby"

	// Detect framework
	if strings.Contains(content, "rails") {
		techStack["framework"] = "Ruby on Rails"
	} else if strings.Contains(content, "sinatra") {
		techStack["framework"] = "Sinatra"
	}

	return techStack
}

func extractFromCargoToml(content string) map[string]string {
	techStack := make(map[string]string)

	techStack["language"] = "Rust"

	// Detect framework
	if strings.Contains(content, "actix-web") {
		techStack["framework"] = "Actix Web"
	} else if strings.Contains(content, "rocket") {
		techStack["framework"] = "Rocket"
	}

	return techStack
}

func extractFromComposerJson(content string) map[string]string {
	techStack := make(map[string]string)

	techStack["language"] = "PHP"

	var composer struct {
		Require map[string]string `json:"require"`
	}

	if err := json.Unmarshal([]byte(content), &composer); err != nil {
		return techStack
	}

	// Detect framework
	if _, ok := composer.Require["laravel/framework"]; ok {
		techStack["framework"] = "Laravel"
	} else if _, ok := composer.Require["symfony/symfony"]; ok {
		techStack["framework"] = "Symfony"
	}

	return techStack
}

// AggregateTechStack aggregates tech stacks from multiple repositories
func AggregateTechStack(repos []*GitHubRepo) map[string]string {
	aggregated := make(map[string]string)
	techCount := make(map[string]int)

	for _, repo := range repos {
		for tech, value := range repo.TechStack {
			if techCount[tech] < techCount[value] || techCount[tech] == 0 {
				aggregated[tech] = value
				techCount[tech]++
			}
		}
	}

	return aggregated
}

// CalculateExperienceLevel calculates experience level based on repositories
func CalculateExperienceLevel(repos []*GitHubRepo, totalStars int) string {
	repoCount := len(repos)

	if repoCount >= 20 && totalStars >= 100 {
		return "Senior"
	} else if repoCount >= 10 && totalStars >= 50 {
		return "Mid"
	} else {
		return "Junior"
	}
}
