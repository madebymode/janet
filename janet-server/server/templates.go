package server

import (
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
)

// TemplateData structures for template rendering
type TemplateData struct {
	Title     string
	BotOnline bool
	Version   string
	ExtraJS   []string
	Data      interface{}
	Error     string
}

// setupTemplates loads and parses HTML templates from embedded filesystem  
func (s *Server) setupTemplates() error {
	s.templates = make(map[string]*template.Template)

	// Load templates from embedded filesystem
	templateFiles := []string{
		"web/templates/app.html",
		"web/templates/index.html", 
		"web/templates/leaderboard.html",
		"web/templates/stats.html",
	}

	for _, file := range templateFiles {
		content, err := s.webFS.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", file, err)
		}

		name := filepath.Base(file)
		name = strings.TrimSuffix(name, filepath.Ext(name))

		// Public templates use base template
		baseTmpl, err := s.webFS.ReadFile("web/templates/base.html")
		if err != nil {
			return fmt.Errorf("failed to read base template: %w", err)
		}
		tmpl, err := template.New("base").Parse(string(baseTmpl))
		if err != nil {
			return fmt.Errorf("failed to parse base template: %w", err)
		}
		tmpl, err = tmpl.Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", file, err)
		}

		s.templates[name] = tmpl
	}

	return nil
}