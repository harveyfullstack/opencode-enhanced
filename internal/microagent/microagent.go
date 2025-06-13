package microagent

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencode-ai/opencode/internal/config"
	"gopkg.in/yaml.v3"
)

const (
	microagentDir = ".opencode/microagents"
)

type Frontmatter struct {
	Triggers []string `yaml:"triggers"`
}

type Microagent struct {
	Frontmatter Frontmatter
	Content     string
	Filepath    string
}

type Finder struct {
	microagents []Microagent
}

func NewFinder() (*Finder, error) {
	finder := &Finder{}
	if err := finder.loadMicroagents(); err != nil {
		return nil, err
	}
	return finder, nil
}

func (f *Finder) loadMicroagents() error {
	cfg := config.Get()
	agentsDir := filepath.Join(cfg.WorkingDir, microagentDir)

	if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.WalkDir(agentsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			agent, err := parseMicroagent(content)
			if err != nil {
				return err
			}
			agent.Filepath = path
			f.microagents = append(f.microagents, agent)
		}
		return nil
	})
}

func parseMicroagent(content []byte) (Microagent, error) {
	var agent Microagent
	parts := bytes.SplitN(content, []byte("---"), 3)

	if len(parts) == 3 {
		if err := yaml.Unmarshal(parts[1], &agent.Frontmatter); err != nil {
			return Microagent{}, err
		}
		agent.Content = string(parts[2])
	} else {
		agent.Content = string(content)
	}

	return agent, nil
}

func (f *Finder) Find(prompt string) []Microagent {
	var matchedAgents []Microagent
	for _, agent := range f.microagents {
		for _, trigger := range agent.Frontmatter.Triggers {
			if strings.Contains(prompt, trigger) {
				matchedAgents = append(matchedAgents, agent)
				break
			}
		}
	}
	return matchedAgents
}