package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	composeLoader "github.com/compose-spec/compose-go/v2/loader"
	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/vshn/k8ify/internal"
	"github.com/vshn/k8ify/pkg/converter"
	"github.com/vshn/k8ify/pkg/ir"
	"github.com/vshn/k8ify/pkg/provider"
	"github.com/vshn/k8ify/pkg/util"
)

var (
	defaultConfig = internal.Config{
		OutputDir: "manifests",
		Env:       "dev",
		Ref:       "",
	}
	modifiedImages   internal.ModifiedImagesFlag
	shellEnvFiles    internal.ShellEnvFilesFlag
	composeFiles     internal.StringSliceFlag
	pflagInitialized = false
)

func InitPflag() {
	// Main() may be called multiple times from tests, hence this kludge
	if !pflagInitialized {
		pflag.Var(&modifiedImages, "modified-image", "Image that has been modified during the build. Can be repeated.")
		pflag.Var(&shellEnvFiles, "shell-env-file", "Shell environment file ('key=value' format) to be used in addition to the current shell environment. Can be repeated.")
		pflag.VarP(&composeFiles, "file", "f", "Compose file to convert instead of the automatic discovery. Can be repeated to merge multiple files, like 'docker compose -f'. Takes precedence over the COMPOSE_FILE environment variable. For each file an environment specific override is also loaded by inserting '-<env>' before the extension (e.g. compose-prod.yml for compose.yml with env prod), so listing only the base file is enough.\n\nThe COMPOSE_FILE environment variable offers the same selection when the flag is not set. It lists one or more files separated by the COMPOSE_PATH_SEPARATOR variable (defaulting to the OS path list separator, ':' on Linux/macOS, ';' on Windows).")
		pflagInitialized = true
	}
}

func main() {
	code := Main(os.Args)
	os.Exit(code)
}

func Main(args []string) int {
	InitPflag()
	// Reset flag state so that repeated invocations (e.g. from tests) don't accumulate values from previous runs.
	modifiedImages.Values = nil
	shellEnvFiles.Values = nil
	composeFiles.Values = nil
	err := pflag.CommandLine.Parse(args[1:])
	if err != nil {
		logrus.Error(err)
		return 1
	}
	plainArgs := pflag.Args()

	config := defaultConfig // this code may run multiple times during testing, thus we can't modify the defaults and must create a copy
	if len(plainArgs) > 0 {
		config.Env = plainArgs[0]
	}
	if len(plainArgs) > 1 {
		config.Ref = plainArgs[1]
	}

	// Load the additional shell environment files first. This merges everything into the existing shell environment
	// (retrievable via os.Environ()) so that variables defined there - e.g. COMPOSE_FILE - are available below.
	for _, shellEnvFile := range shellEnvFiles.Values {
		err := godotenv.Load(shellEnvFile)
		if err != nil {
			logrus.Errorf("Loading dotfiles: %s", err)
			return 1
		}
	}

	if len(config.ConfigFiles) == 0 {
		composeFileEnv := os.Getenv("COMPOSE_FILE")
		switch {
		case len(composeFiles.Values) > 0:
			config.ConfigFiles = composeFiles.Values
		case composeFileEnv != "":
			separator := os.Getenv("COMPOSE_PATH_SEPARATOR")
			if separator == "" {
				separator = string(os.PathListSeparator)
			}
			config.ConfigFiles = strings.Split(composeFileEnv, separator)
		default:
			config.ConfigFiles = []string{"compose.yml", "docker-compose.yml"}
		}

		config.ConfigFiles = appendDeduplicatedEnvOverridesInOrder(config.ConfigFiles, config.Env)
	}

	composeConfigFiles := []composeTypes.ConfigFile{}
	for _, configFile := range config.ConfigFiles {
		if _, err := os.Stat(configFile); err == nil {
			composeConfigFiles = append(composeConfigFiles, composeTypes.ConfigFile{
				Filename: configFile,
			})
		}
	}
	env := util.GetEnv()
	for _, key := range []string{"_ref_", "_secretRef_", "_fieldRef_"} {
		if _, ok := env[key]; ok {
			logrus.Errorf("The environment variable '%s' must not be defined as it is needed for internal purposes", key)
			return 1
		}
	}
	env["_ref_"] = converter.SecretRefMagic
	env["_secretRef_"] = converter.SecretRefMagic
	env["_fieldRef_"] = converter.FieldRefMagic
	configDetails := composeTypes.ConfigDetails{
		ConfigFiles: composeConfigFiles,
		Environment: env,
	}
	project, err := composeLoader.LoadWithContext(context.Background(), configDetails, func(opts *composeLoader.Options) { opts.SetProjectName("k8ify", true) })
	if err != nil {
		logrus.Errorf("Loading compose configuration: %s", err)
		return 1
	}

	inputs := ir.FromCompose(project)
	internal.ComposeServicePrecheck(inputs)
	internal.VolumesPrecheck(inputs)
	internal.DomainLengthPrecheck(inputs)

	objects := converter.Objects{}

	for _, service := range inputs.Services {
		objects = objects.Append(converter.ComposeServiceToK8s(config.Ref, service, inputs.Volumes, inputs.TargetCfg))
	}

	forceRestartAnnotation := make(map[string]string)
	forceRestartAnnotation["k8ify.restart-trigger"] = fmt.Sprintf("%d", time.Now().Unix())
	converter.PatchDeployments(objects.Deployments, modifiedImages.Values, objects.Secrets, forceRestartAnnotation)
	converter.PatchStatefulSets(objects.StatefulSets, modifiedImages.Values, objects.Secrets, forceRestartAnnotation)

	objects = provider.PatchEncryptedVolumeSchemeAppuioCloudscale(inputs.TargetCfg, config, objects)

	err = internal.WriteManifests(config.OutputDir, objects)
	if err != nil {
		logrus.Errorf("Writing manifests: %s", err)
		return 1
	}

	return 0
}

func appendDeduplicatedEnvOverridesInOrder(files []string, env string) []string {
	seen := make(map[string]struct{})
	var out []string

	add := func(f string) {
		if _, ok := seen[f]; !ok {
			seen[f] = struct{}{}
			out = append(out, f)
		}
	}

	for _, f := range files {
		add(f)
		add(envOverrideName(f, env))
	}
	return out
}

func envOverrideName(file, env string) string {
	ext := filepath.Ext(file)
	return strings.TrimSuffix(file, ext) + "-" + env + ext
}
