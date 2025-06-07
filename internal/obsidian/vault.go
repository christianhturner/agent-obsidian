package obsidian

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Vault struct {
	config *Config
}

type Note struct {
	Path         string    `json:"path"`
	Name         string    `json:"name"`
	Title        string    `json:"title"`
	RelativePath string    `json:"relative_path"`
	CreatedAt    time.Time `json:"created_at"`
	ModifiedAt   time.Time `json:"modified_at"`
	Tags         []string  `json:"tags"`
	WordCount    int       `json:"word_count"`
}

func NewVault(config *Config) (*Vault, error) {
	// Validate vault path with smart error handling
	if err := validateVaultPath(config.VaultPath); err != nil {
		return nil, err
	}

	return &Vault{config: config}, nil
}

func validateVaultPath(vaultPath string) error {
	if _, err := os.Stat(vaultPath); err != nil {
		if os.IsNotExist(err) {
			// Try to provide helpful suggestions
			parent := filepath.Dir(vaultPath)
			if _, parentErr := os.Stat(parent); parentErr == nil {
				// Parent exists, suggest alternatives
				return suggestAlternatives(parent, filepath.Base(vaultPath))
			}
			return fmt.Errorf("vault path does not exist: %s", vaultPath)
		}
		return fmt.Errorf("cannot access vault path: %w", err)
	}
	return nil
}

func suggestAlternatives(parentDir, targetName string) error {
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return fmt.Errorf("vault path does not exist and cannot read parent directory: %s", parentDir)
	}

	var directories []string
	for _, entry := range entries {
		if entry.IsDir() {
			directories = append(directories, entry.Name())
		}
	}

	if len(directories) == 0 {
		return fmt.Errorf("vault path does not exist: %s (no directories found in parent)",
			filepath.Join(parentDir, targetName))
	}

	return fmt.Errorf("vault path does not exist: %s\nDid you mean one of these?\n  %s",
		filepath.Join(parentDir, targetName),
		strings.Join(directories, "\n  "))
}

func (v *Vault) ListNotes() ([]Note, error) {
	var notes []Note

	err := filepath.Walk(v.config.VaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Check if this directory should be ignored
			for _, ignored := range v.config.IgnoredDirectories {
				if strings.Contains(path, ignored) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check file extension
		ext := filepath.Ext(path)
		validExt := false
		for _, allowedExt := range v.config.FileExtensions {
			if ext == allowedExt {
				validExt = true
				break
			}
		}

		if !validExt {
			return nil
		}

		// Parse note details
		note, err := v.parseNote(path, info)
		if err != nil {
			// Log error but continue processing other files
			fmt.Fprintf(os.Stderr, "Warning: failed to parse note %s: %v\n", path, err)
			return nil
		}

		notes = append(notes, note)
		return nil
	})

	return notes, err
}

func (v *Vault) parseNote(path string, info os.FileInfo) (Note, error) {
	// Create relative path from vault root
	relPath, err := filepath.Rel(v.config.VaultPath, path)
	if err != nil {
		return Note{}, err
	}

	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return Note{}, err
	}

	contentStr := string(content)

	// Get file times (creation time is tricky, using mod time for both for now)
	modTime := info.ModTime()
	createTime := modTime // We'll improve this later if needed

	note := Note{
		Path:         path,
		Name:         strings.TrimSuffix(info.Name(), filepath.Ext(path)),
		RelativePath: relPath,
		CreatedAt:    createTime,
		ModifiedAt:   modTime,
		Title:        extractTitle(contentStr),
		Tags:         extractTags(contentStr),
		WordCount:    countWords(contentStr),
	}

	return note, nil
}

func extractTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return "" // No title found
}

func extractTags(content string) []string {
	var tags []string

	// Look for hashtags (#tag)
	hashtagRegex := regexp.MustCompile(`#([a-zA-Z0-9_-]+)`)
	matches := hashtagRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			tags = append(tags, match[1])
		}
	}

	// Look for YAML frontmatter tags
	if strings.HasPrefix(content, "---") {
		frontmatterEnd := strings.Index(content[3:], "---")
		if frontmatterEnd != -1 {
			frontmatter := content[3 : frontmatterEnd+3]
			yamlTags := extractYAMLTags(frontmatter)
			tags = append(tags, yamlTags...)
		}
	}

	// Remove duplicates
	return removeDuplicates(tags)
}

func extractYAMLTags(frontmatter string) []string {
	var tags []string
	scanner := bufio.NewScanner(strings.NewReader(frontmatter))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "tags:") {
			// Handle single line: tags: [tag1, tag2]
			if strings.Contains(line, "[") {
				tagStr := strings.TrimSpace(strings.TrimPrefix(line, "tags:"))
				tagStr = strings.Trim(tagStr, "[]")
				for _, tag := range strings.Split(tagStr, ",") {
					tag = strings.TrimSpace(strings.Trim(tag, "\"'"))
					if tag != "" {
						tags = append(tags, tag)
					}
				}
			}
		} else if strings.HasPrefix(line, "- ") && len(tags) > 0 {
			// Handle multi-line tags
			tag := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			tag = strings.Trim(tag, "\"'")
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	return tags
}

func countWords(content string) int {
	// Remove frontmatter
	if strings.HasPrefix(content, "---") {
		frontmatterEnd := strings.Index(content[3:], "---")
		if frontmatterEnd != -1 {
			content = content[frontmatterEnd+6:]
		}
	}

	// Simple word count
	words := strings.Fields(content)
	return len(words)
}

func removeDuplicates(tags []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, tag := range tags {
		if !keys[tag] {
			keys[tag] = true
			result = append(result, tag)
		}
	}

	return result
}
