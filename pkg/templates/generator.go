package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GenerateFromTemplate creates a complete project from the specified template
func GenerateFromTemplate(templateName, targetDir string) error {
	// Validate template name
	template, exists := getTemplateByName(templateName)
	if !exists {
		return fmt.Errorf("unknown template '%s'. Available templates: go, python, node", templateName)
	}

	// Get and sanitize project name from target directory
	projectName := sanitizeProjectName(filepath.Base(targetDir))

	// Check for file conflicts before creating anything
	if err := checkFileConflicts(template.Files, targetDir); err != nil {
		return err
	}

	// Create all template files
	for _, file := range template.Files {
		// Replace placeholder project name in content
		content := strings.ReplaceAll(file.Content, "{{PROJECT_NAME}}", projectName)

		// Create full file path
		filePath := filepath.Join(targetDir, file.Path)

		// Create directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", file.Path, err)
		}

		// Write file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", file.Path, err)
		}
	}

	fmt.Printf("✅ Generated %s project '%s' with %d files\n", templateName, projectName, len(template.Files))
	fmt.Printf("Next steps:\n")
	fmt.Printf("  cd %s\n", targetDir)
	fmt.Printf("  reactor up\n")

	return nil
}

// sanitizeProjectName applies consistent sanitization rules for all package managers
func sanitizeProjectName(name string) string {
	if name == "" {
		return "my-app"
	}

	// Step 1: Convert to lowercase
	sanitized := strings.ToLower(name)

	// Step 2: Replace spaces and special characters with hyphens
	// Keep only alphanumeric characters and hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	sanitized = reg.ReplaceAllString(sanitized, "-")

	// Step 3: Collapse multiple consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	sanitized = reg.ReplaceAllString(sanitized, "-")

	// Step 4: If starts with number or hyphen, prefix with "app-"
	if len(sanitized) > 0 && (sanitized[0] >= '0' && sanitized[0] <= '9' || sanitized[0] == '-') {
		sanitized = "app-" + sanitized
	}

	// Step 5: Remove leading and trailing hyphens
	sanitized = strings.Trim(sanitized, "-")

	// Final fallback if somehow empty
	if sanitized == "" {
		sanitized = "my-app"
	}

	return sanitized
}

// checkFileConflicts verifies that template files won't overwrite existing files
func checkFileConflicts(templateFiles []TemplateFile, targetDir string) error {
	var conflicts []string

	for _, file := range templateFiles {
		filePath := filepath.Join(targetDir, file.Path)
		if _, err := os.Stat(filePath); err == nil {
			conflicts = append(conflicts, file.Path)
		}
	}

	if len(conflicts) > 0 {
		return fmt.Errorf("template files would conflict with existing files: %s\n"+
			"Please remove these files or run the command in a different directory",
			strings.Join(conflicts, ", "))
	}

	return nil
}
