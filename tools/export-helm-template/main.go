// Export a built-in Helm template without accessing a database or cluster.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ChenJustin666/axiops/internal/cicd/templatelib"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
)

func main() {
	key := flag.String("key", "helm-sinozo-java-v1", "built-in Helm template key")
	output := flag.String("output", "local/dev/helm-templates", "artifact directory")
	flag.Parse()
	if err := export(*key, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func export(key, output string) error {
	for _, template := range templatelib.Defaults() {
		if template.Kind != "helm" || template.BuiltinKey != key {
			continue
		}
		if err := templatelib.Validate(template); err != nil {
			return err
		}
		files := []*loader.BufferedFile{}
		for _, file := range template.Files {
			files = append(files, &loader.BufferedFile{Name: file.Path, Data: []byte(file.Content)})
		}
		c, err := loader.LoadFiles(files)
		if err != nil {
			return err
		}
		if err = os.MkdirAll(output, 0755); err != nil {
			return err
		}
		archive, err := chartutil.Save(c, output)
		if err != nil {
			return err
		}
		// The payload creates an independent user template, not a new built-in identity.
		template.BuiltinKey = ""
		payload, err := json.MarshalIndent(template, "", "  ")
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(output, c.Name()+"-template.json"), payload, 0644); err != nil {
			return err
		}
		// Loading the generated archive also checks the exported file layout.
		if _, err = loader.LoadFile(archive); err != nil {
			return err
		}
		fmt.Println(archive)
		return nil
	}
	return fmt.Errorf("Helm template %q not found", key)
}
