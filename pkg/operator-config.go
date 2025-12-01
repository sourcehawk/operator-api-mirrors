package pkg

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

type GitRepository string

func (r GitRepository) URL() string {
	return fmt.Sprintf("https://%s", string(r))
}

type DirectoryPath string

func (d DirectoryPath) GetPaths(root string) ([]string, error) {
	pattern := filepath.Join(root, string(d))

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob failed for pattern %q: %w", pattern, err)
	}

	return matches, nil
}

type Dependency struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type OperatorConfig struct {
	Slug           string          `yaml:"slug"`
	Repo           GitRepository   `yaml:"repo"`
	CurrentVersion string          `yaml:"currentVersion"`
	GoModPath      string          `yaml:"goModPath"`
	APIs           []DirectoryPath `yaml:"apiPaths"`
	OverwriteDeps  []Dependency    `yaml:"overwriteDependencies"`
}

func (o OperatorConfig) Mirror(mirrorRootPath string, moduleRoot string) error {
	destDir := o.directory(mirrorRootPath)

	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("failed to remove %s: %w", destDir, err)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", destDir, err)
	}

	sourceDir, err := o.clone()
	if err != nil {
		return err
	}

	for _, api := range o.APIs {
		paths, err := api.GetPaths(sourceDir)
		if err != nil {
			return err
		}

		err = o.copy(paths, sourceDir, destDir)
		if err != nil {
			return err
		}
	}

	mirrorRootPathStripped := strings.TrimPrefix(mirrorRootPath, ".")
	mirrorRootPathStripped = strings.TrimPrefix(mirrorRootPathStripped, "/")
	modulePath := fmt.Sprintf("%s/%s", moduleRoot, mirrorRootPathStripped)

	err = o.rewriteInternalImportsAndCopy(sourceDir, destDir, modulePath)
	if err != nil {
		return err
	}

	err = o.createGoMod(sourceDir, destDir, modulePath)
	if err != nil {
		return err
	}

	_ = os.RemoveAll(sourceDir)
	return nil
}

func (o OperatorConfig) directory(mirrorRootPath string) string {
	return fmt.Sprintf("%s/%s/%s", mirrorRootPath, o.Slug, o.CurrentVersion)
}

func (o OperatorConfig) clone() (string, error) {
	tmp, err := os.MkdirTemp("", fmt.Sprintf("%s-%s-", o.Slug, o.CurrentVersion))
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", o.CurrentVersion, o.Repo.URL(), tmp)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		_ = os.RemoveAll(tmp)
		return "", fmt.Errorf("failed to clone %s: %w", o.Repo, err)
	}

	return tmp, nil
}

// Copy copies all relevant files from the given source directories to the destination under the same relative path.
func (o OperatorConfig) copy(sourcePaths []string, sourceRoot, destinationRoot string) error {
	for _, sourcePath := range sourcePaths {
		rel, err := filepath.Rel(sourceRoot, sourcePath)
		if err != nil {
			return fmt.Errorf("failed to make %q relative to %q: %w", sourcePath, sourceRoot, err)
		}

		destPath := filepath.Join(destinationRoot, rel)
		log.Printf("Copying %s -> %s", sourcePath, destPath)
		if err := copyGoFiles(sourcePath, destPath); err != nil {
			return fmt.Errorf("copy %s -> %s: %w", sourcePath, destPath, err)
		}
	}

	return nil
}

func (o OperatorConfig) rewriteInternalImportsAndCopy(sourceDir, destDir, moduleRoot string) error {
	upstreamModule, err := getUpstreamModulePath(sourceDir)
	if err != nil {
		return err
	}

	mirrorModule := fmt.Sprintf("%s/%s/%s", moduleRoot, o.Slug, o.CurrentVersion)

	seenImports := map[string]struct{}{}

	for {
		internalImports, err := collectInternalImports(destDir, upstreamModule)
		if err != nil {
			return err
		}

		var toCopy []string
		for _, imp := range internalImports {
			if _, ok := seenImports[imp]; ok {
				continue
			}
			seenImports[imp] = struct{}{}
			toCopy = append(toCopy, imp)
		}

		if len(toCopy) == 0 {
			// No new upstream imports found -> we've reached closure
			break
		}

		if err := copyInternalPackages(toCopy, upstreamModule, sourceDir, destDir); err != nil {
			return err
		}
	}

	// Now that all needed upstream packages exist locally, rewrite imports once
	if err := rewriteImports(destDir, upstreamModule, mirrorModule); err != nil {
		return err
	}

	return nil
}

func (o OperatorConfig) createGoMod(sourceDir string, destDir string, moduleRoot string) error {
	mirrorModulePath := fmt.Sprintf("%s/%s/%s", moduleRoot, o.Slug, o.CurrentVersion)
	upstreamGoMod := filepath.Join(sourceDir, o.GoModPath)

	data, err := os.ReadFile(upstreamGoMod)
	if err != nil {
		return fmt.Errorf("read upstream go.mod: %w", err)
	}

	upstream, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return fmt.Errorf("parse upstream go.mod: %w", err)
	}

	// Create new mod file for the mirror module
	newMod := &modfile.File{}
	if err := newMod.AddModuleStmt(mirrorModulePath); err != nil {
		return fmt.Errorf("add module stmt: %w", err)
	}

	// Set Go version (or fall back to 1.22)
	goVersion := "1.22"
	if upstream.Go != nil && upstream.Go.Version != "" {
		goVersion = upstream.Go.Version
	}
	if err := newMod.AddGoStmt(goVersion); err != nil {
		return fmt.Errorf("add go stmt: %w", err)
	}

	// Copy require directives only (no replace/exclude for now)
	for _, r := range upstream.Require {
		if err := newMod.AddRequire(r.Mod.Path, r.Mod.Version); err != nil {
			return fmt.Errorf("failed to add require %s %s: %w", r.Mod.Path, r.Mod.Version, err)
		}
	}

	// Write new go.mod
	goModPath := filepath.Join(destDir, "go.mod")
	formatted, err := newMod.Format()
	if err != nil {
		return fmt.Errorf("format new go.mod: %w", err)
	}

	if err := os.WriteFile(goModPath, formatted, 0o644); err != nil {
		return fmt.Errorf("write go.mod: %w", err)
	}

	// Apply replace-only overrides
	for _, dep := range o.OverwriteDeps {
		cmd := exec.Command(
			"go", "mod", "edit",
			fmt.Sprintf("-replace=%[1]s=%[1]s@%[2]s", dep.Name, dep.Version),
		)
		cmd.Dir = destDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go mod edit -replace %s@%s failed: %w", dep.Name, dep.Version, err)
		}
	}

	if err := tidy(destDir); err != nil {
		return err
	}

	return nil
}
