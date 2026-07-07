package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LessonStep is one runnable chunk in a tutorial file.
type LessonStep struct {
	Hint string
	Code string
}

// Lesson is a numbered interactive tutorial.
type Lesson struct {
	Number  int
	Title   string
	File    string
	Steps   []LessonStep
	Summary string
}

// LessonProgress tracks the active guided lesson.
type LessonProgress struct {
	Lesson    Lesson
	StepIndex int
}

// DiscoverLessons loads tutorial-*.funny files from dir.
func DiscoverLessons(dir string) ([]Lesson, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "tutorial-") && strings.HasSuffix(name, ".funny") {
			files = append(files, name)
		}
	}
	sort.Strings(files)
	lessons := make([]Lesson, 0, len(files))
	for i, name := range files {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		lesson, err := parseLessonFile(i+1, path, string(data))
		if err != nil {
			return nil, err
		}
		lessons = append(lessons, lesson)
	}
	return lessons, nil
}

func parseLessonFile(number int, path, content string) (Lesson, error) {
	title := fmt.Sprintf("Lesson %d", number)
	summary := ""
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "#") {
			break
		}
		text := strings.TrimSpace(strings.TrimPrefix(t, "#"))
		if strings.HasPrefix(text, "Tutorial ") {
			title = text
		} else if summary == "" && text != "" {
			summary = text
		}
	}
	steps := parseLessonSteps(content)
	if len(steps) == 0 {
		return Lesson{}, fmt.Errorf("lesson %s has no runnable steps", path)
	}
	return Lesson{
		Number:  number,
		Title:   title,
		File:    path,
		Steps:   steps,
		Summary: summary,
	}, nil
}

func parseLessonSteps(content string) []LessonStep {
	var steps []LessonStep
	for _, block := range strings.Split(content, "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		var hint []string
		var code []string
		for _, line := range strings.Split(block, "\n") {
			t := strings.TrimSpace(line)
			if t == "" {
				continue
			}
			if strings.HasPrefix(t, "#") {
				hint = append(hint, strings.TrimSpace(strings.TrimPrefix(t, "#")))
			} else {
				code = append(code, line)
			}
		}
		if len(code) == 0 {
			continue
		}
		steps = append(steps, LessonStep{
			Hint: strings.Join(hint, "\n"),
			Code: strings.Join(code, "\n"),
		})
	}
	return steps
}

func (p *LessonProgress) current() (LessonStep, bool) {
	if p == nil || p.StepIndex < 0 || p.StepIndex >= len(p.Lesson.Steps) {
		return LessonStep{}, false
	}
	return p.Lesson.Steps[p.StepIndex], true
}

func (p *LessonProgress) advance() bool {
	if p == nil {
		return false
	}
	p.StepIndex++
	return p.StepIndex < len(p.Lesson.Steps)
}
