package internal

import (
	"fmt"
)

type ModifiedImagesFlag struct {
	Values []string
}

func (f *ModifiedImagesFlag) Set(value string) error {
	f.Values = append(f.Values, value)
	return nil
}

func (f *ModifiedImagesFlag) String() string {
	return fmt.Sprintf("%v", f.Values)
}

func (f *ModifiedImagesFlag) Type() string {
	return "myapp:latest"
}

type ShellEnvFilesFlag struct {
	Values []string
}

func (f *ShellEnvFilesFlag) Set(value string) error {
	f.Values = append(f.Values, value)
	return nil
}

func (f *ShellEnvFilesFlag) String() string {
	return fmt.Sprintf("%v", f.Values)
}

func (f *ShellEnvFilesFlag) Type() string {
	return ".env"
}

type ProviderFlag struct {
	value string
}

func (f *ProviderFlag) Set(value string) error {
	f.value = value
	return nil
}

func (f *ProviderFlag) String() string {
	return f.value
}

func (f *ProviderFlag) Type() string {
	return "appuio-cloudscale"
}
