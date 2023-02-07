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
