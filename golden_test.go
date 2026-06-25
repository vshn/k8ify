package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	GOLDEN_PATH = "tests/golden"
)

var (
	ext = regexp.MustCompile(`\.ya?ml$`)
)

type Instance struct {
	Environments map[string]Environment `yaml:"environments"`
	// Flag is an optional top-level config whose params are merged into every environment.
	Flag Environment `yaml:"flag"`
	// Env is an optional top-level config whose vars are merged into every environment
	// (environment-specific vars take precedence over these).
	Env Environment `yaml:"env"`
}

type Environment struct {
	Refs   []string          `yaml:"refs"`
	Vars   map[string]string `yaml:"vars"`
	Params []string          `yaml:"params"`
}

func init() {
	logrus.SetOutput(io.Discard)
}

// GetRefs returns the list of refs defined in the test spec, or just an empty
// string if no refs have been defined.
func (e *Environment) GetRefs() []string {
	if len(e.Refs) < 1 {
		return []string{""}
	}

	return e.Refs
}

func TestGolden(t *testing.T) {
	instances := findInstances()
	for _, i := range instances {
		t.Run(i, func(t *testing.T) {
			testInstance(t, i)
		})
	}
}

func findInstances() []string {
	c, err := os.ReadDir(GOLDEN_PATH)
	check(err, "finding instances")

	instances := make([]string, 0)
	ext := regexp.MustCompile(`\.ya?ml$`)
	for _, e := range c {
		if ext.MatchString(e.Name()) && !e.IsDir() {
			instances = append(instances, e.Name())
		}
	}

	return instances
}

func testInstance(t *testing.T, instanceFile string) {
	f, err := os.Open(filepath.Join(GOLDEN_PATH, instanceFile))
	check(err, "reading golden test definition")

	instance := ext.ReplaceAllString(instanceFile, "")
	i := Instance{}
	check(yaml.NewDecoder(f).Decode(&i), "decoding golden test definition")

	old, err := os.Getwd()
	check(err, "determining current working directory")

	root := filepath.Join(GOLDEN_PATH, instance)
	check(os.Chdir(root), "changing working directory")

	for name, env := range i.Environments {
		// Merge top-level flag params and env vars into the environment.
		// Environment-specific values take precedence over top-level ones.
		merged := Environment{
			Refs:   env.Refs,
			Params: append(append([]string{}, i.Flag.Params...), env.Params...),
			Vars:   make(map[string]string, len(i.Env.Vars)+len(env.Vars)),
		}
		for k, v := range i.Env.Vars {
			merged.Vars[k] = v
		}
		for k, v := range env.Vars {
			merged.Vars[k] = v
		}
		t.Run(name, func(t *testing.T) {
			testEnvironment(t, name, merged)
		})
	}

	check(os.Chdir(old), "resetting working directory")
}

func testEnvironment(t *testing.T, envName string, env Environment) {
	for k, v := range env.Vars {
		t.Setenv(k, v)
	}

	for _, ref := range env.GetRefs() {
		t.Run(ref, func(t *testing.T) {
			args := []string{"k8ify", envName, ref}
			args = append(args, env.Params...)
			fmt.Printf("Running %v", args)
			var logs bytes.Buffer
			logrus.SetOutput(&logs)
			if c := Main(args); c != 0 {
				t.Errorf("k8ify exited with code %v while compiling with args '%v'\n%s", c, args, logs.String())
			}

			cmd := exec.Command("git", "diff", "--exit-code", "--minimal", "--", "manifests/")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("error from git diff: %v", err)
				fmt.Println(string(out))
			}
		})
	}
}

func check(err error, context string) {
	if err != nil {
		_, cf, cl, _ := runtime.Caller(0)
		cf = filepath.Base(cf)
		log.Fatalf("%s:%d: Error %s: %v", cf, cl, context, err)
	}
}
