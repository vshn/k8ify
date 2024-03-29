package internal

import (
	"fmt"
)

type ModifiedImagesFlag struct {
	Values []string
}

func (this *ModifiedImagesFlag) Set(value string) error {
	this.Values = append(this.Values, value)
	return nil
}

func (this *ModifiedImagesFlag) String() string {
	return fmt.Sprintf("%v", this.Values)
}

func (this *ModifiedImagesFlag) Type() string {
	return "myapp:latest"
}

type ShellEnvFilesFlag struct {
	Values []string
}

func (this *ShellEnvFilesFlag) Set(value string) error {
	this.Values = append(this.Values, value)
	return nil
}

func (this *ShellEnvFilesFlag) String() string {
	return fmt.Sprintf("%v", this.Values)
}

func (this *ShellEnvFilesFlag) Type() string {
	return ".env"
}

type ProviderFlag struct {
	value string
}

func (this *ProviderFlag) Set(value string) error {
	this.value = value
	return nil
}

func (this *ProviderFlag) String() string {
	return this.value
}

func (this *ProviderFlag) Type() string {
	return "appuio-cloudscale"
}
