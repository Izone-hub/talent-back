package service

// deriveCategories maps a user's GitHub top languages to the job categories
// shown on the admin Users page. A user can belong to several categories
// (e.g. a fullstack developer is both backend and frontend), so the result is
// a list. Empty when nothing matches.
func deriveCategories(topLanguages []string) []string {
	backendLangs := map[string]bool{
		"Go": true, "Python": true, "Java": true, "Rust": true, "C": true,
		"C++": true, "C#": true, "Ruby": true, "PHP": true, "Elixir": true,
		"Scala": true,
	}
	frontendLangs := map[string]bool{
		"JavaScript": true, "TypeScript": true, "HTML": true, "CSS": true,
		"Vue": true, "Svelte": true,
	}
	mobileLangs := map[string]bool{
		"Dart": true, "Kotlin": true, "Swift": true,
	}
	devopsLangs := map[string]bool{
		"Shell": true, "Dockerfile": true, "HCL": true, "Makefile": true,
	}

	var hasBackend, hasFrontend, hasMobile, hasDevops bool
	for _, lang := range topLanguages {
		if backendLangs[lang] {
			hasBackend = true
		}
		if frontendLangs[lang] {
			hasFrontend = true
		}
		if mobileLangs[lang] {
			hasMobile = true
		}
		if devopsLangs[lang] {
			hasDevops = true
		}
	}

	cats := make([]string, 0, 5)
	if hasBackend {
		cats = append(cats, "backend_developer")
	}
	if hasFrontend {
		cats = append(cats, "frontend_developer")
	}
	if hasBackend && hasFrontend {
		cats = append(cats, "full_stack_developer")
	}
	if hasMobile {
		cats = append(cats, "mobile_developer")
	}
	if hasDevops {
		cats = append(cats, "system_architect")
	}
	return cats
}