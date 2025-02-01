package shape

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"text/template"
)

//go:embed messages/*.json
var messageFS embed.FS

var (
	messageCatalogs     map[string]map[string]*template.Template
	messageCatalogsOnce sync.Once
	messageCatalogsErr  error
)

func loadMessageCatalogs() {
	messageCatalogsOnce.Do(func() {
		entries, err := messageFS.ReadDir("messages")
		if err != nil {
			messageCatalogsErr = err
			return
		}
		catalogs := make(map[string]map[string]*template.Template, len(entries))
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			locale := strings.TrimSuffix(entry.Name(), ".json")
			data, err := messageFS.ReadFile("messages/" + entry.Name())
			if err != nil {
				messageCatalogsErr = err
				return
			}
			var raw map[string]string
			if err := json.Unmarshal(data, &raw); err != nil {
				messageCatalogsErr = err
				return
			}
			parsed := make(map[string]*template.Template, len(raw))
			for key, text := range raw {
				tmpl, err := template.New(locale + ":" + key).Option("missingkey=zero").Parse(text)
				if err != nil {
					messageCatalogsErr = err
					return
				}
				parsed[key] = tmpl
			}
			catalogs[locale] = parsed
		}
		messageCatalogs = catalogs
	})
}

// Localize rewrites ValidationError issue messages for lang. Non-validation
// errors are returned unchanged. Unknown locales fall back to English catalogs.
func Localize(err error, lang string) error {
	return localizeError(normalizeLocale(lang), err)
}

// LocalizeContext rewrites ValidationError issue messages using the locale
// resolved from ctx and SetLanguage.
func LocalizeContext(ctx context.Context, err error) error {
	return localizeError(resolveLocale(ctx), err)
}

func localizeError(lang string, err error) error {
	if err == nil {
		return nil
	}
	var validation *ValidationError
	if !errors.As(err, &validation) || validation == nil {
		return err
	}
	return &ValidationError{Issues: localizeIssues(lang, validation.Issues)}
}

func localizeIssues(lang string, issues []Issue) []Issue {
	if len(issues) == 0 {
		return nil
	}
	loadMessageCatalogs()
	result := make([]Issue, len(issues))
	for index, issue := range issues {
		result[index] = localizeIssue(lang, issue)
	}
	return result
}

func localizeIssue(lang string, issue Issue) Issue {
	if issue.key == "" && issue.Message != "" {
		return issue
	}
	key := issue.key
	if key == "" {
		key = issue.Code
	}
	if rendered, ok := renderMessage(lang, key, issue); ok {
		issue.Message = rendered
		return issue
	}
	if lang != "en" {
		if rendered, ok := renderMessage("en", key, issue); ok {
			issue.Message = rendered
			return issue
		}
	}
	return issue
}

func renderMessage(lang, key string, issue Issue) (string, bool) {
	if messageCatalogsErr != nil || messageCatalogs == nil {
		return "", false
	}
	catalog := messageCatalogs[lang]
	if catalog == nil {
		return "", false
	}
	tmpl := catalog[key]
	if tmpl == nil {
		return "", false
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, issue); err != nil || buf.Len() == 0 {
		return "", false
	}
	return buf.String(), true
}

// RenderIssue fills Message from the locale catalog. Custom issues that already
// carry a Message and no catalog key are returned unchanged.
func RenderIssue(issue Issue, lang string) Issue {
	return localizeIssue(normalizeLocale(lang), issue)
}

// TruncatedIssuesIssue is the terminal issue used when issue lists are capped.
func TruncatedIssuesIssue() Issue {
	return tooManyIssues()
}

// keyedIssue builds a built-in issue whose Message is resolved from catalogs.
func keyedIssue(code, key string, expected, received any) Issue {
	return Issue{Code: code, Expected: expected, Received: received, key: key}
}
