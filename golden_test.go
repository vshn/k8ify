package main

import (
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
}

type Environment struct {
	Refs []string          `yaml:"refs"`
	Vars map[string]string `yaml:"vars"`
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
		t.Run(name, func(t *testing.T) {
			testEnvironment(t, name, env)
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
			fmt.Printf("Running %v", args)
			if c := Main(args); c != 0 {
				t.Errorf("k8ify exited with code %v while compiling with args %v", c, args)
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
