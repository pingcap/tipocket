package scaffolds

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pingcap/tipocket/pkg/scaffolds/file"
	"github.com/pingcap/tipocket/pkg/scaffolds/model"
)

// Scaffold uses templates to scaffold new files
type Scaffold interface {
	// Execute writes to disk the provided files
	Execute(*model.Universe, ...file.Builder) error
}

// scaffold implements Scaffold interface
type scaffold struct{}

// NewScaffold creates an new scaffold
func NewScaffold() Scaffold {
	return &scaffold{}
}

func (s *scaffold) Execute(universe *model.Universe, files ...file.Builder) error {
	universe.Files = make(map[string]*file.File, len(files))
	for _, f := range files {
		if err := f.Validate(); err != nil {
			return err
		}

		if t, ok := f.(file.Template); ok {
			if err := s.buildFileModel(t, universe.Files); err != nil {
				return err
			}
		}
		if i, ok := f.(file.Inserter); ok {
			if err := s.updateFileModel(i, universe.Files); err != nil {
				return nil
			}
		}
	}
	// Persist the files to disk
	for _, f := range universe.Files {
		if err := s.writeFile(f); err != nil {
			return err
		}
	}
	return nil
}

func (s *scaffold) buildFileModel(t file.Template, models map[string]*file.File) error {
	err := t.SetTemplateDefaults()
	if err != nil {
		return err
	}
	if _, found := models[t.GetPath()]; found {
		return fmt.Errorf("model already exist: %s", t.GetPath())
	}
	m := &file.File{Path: t.GetPath(), IfExistsAction: t.GetIfExistsAction()}
	m.Content, err = buildTemplate(t)
	if err != nil {
		return err
	}
	models[m.Path] = m
	return nil
}

func (s *scaffold) updateFileModel(i file.Inserter, models map[string]*file.File) error {
	m, err := s.loadModelFromFile(i)
	if err != nil {
		return err
	}
	m.Content, err = insertCodeFragments(m.Content, i.GetCodeFragments())
	if err != nil {
		return err
	}
	models[i.GetPath()] = m
	return nil
}

func (s *scaffold) writeFile(f *file.File) error {
	var (
		exist = false
		err   error
	)

	_, err = os.Stat(f.Path)
	if err == nil {
		exist = true
	} else if !os.IsNotExist(err) {
		return err
	}
	if exist {
		switch f.IfExistsAction {
		case file.IfExistsActionError:
			return fmt.Errorf("file already exists: %s", f.Path)
		case file.IfExistsActionOverwrite:
		}
	}
	file, err := s.createFileIfNotExist(f)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write([]byte(f.Content))
	return err
}

func (s *scaffold) createFileIfNotExist(f *file.File) (*os.File, error) {
	dir := filepath.Dir(f.Path)
	_, err := os.Stat(dir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}
	file, err := os.OpenFile(f.Path, os.O_CREATE|os.O_WRONLY, 0644)
	return file, err
}

func (s *scaffold) loadModelFromFile(i file.Inserter) (*file.File, error) {
	if _, err := os.Stat(i.GetPath()); err != nil {
		return nil, err
	}
	b, err := ioutil.ReadFile(i.GetPath())
	if err != nil {
		return nil, err
	}
	return &file.File{
		Path:           i.GetPath(),
		Content:        string(b),
		IfExistsAction: i.GetIfExistsAction(),
	}, nil
}

func buildTemplate(t file.Template) (string, error) {
	temp, err := template.New(fmt.Sprintf("%T", t)).Parse(t.GetBody())
	if err != nil {
		return "", err
	}
	out := &bytes.Buffer{}
	if err := temp.Execute(out, t); err != nil {
		return "", err
	}
	return out.String(), nil
}

func insertCodeFragments(content string, fragmentMaps map[file.Marker]file.CodeFragment) (string, error) {
	out := new(bytes.Buffer)

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()

		for marker, codeFragments := range fragmentMaps {
			if file.Marker(strings.TrimSpace(line)) == marker {
				for _, codeFragment := range codeFragments {
					_, _ = out.WriteString(codeFragment)
				}
			}
		}
		out.WriteString(line + "\n")
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return out.String(), nil
}
