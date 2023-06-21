package dalek

import (
	"fmt"
	"strings"
)

type Options struct {
	Prefix                         string
	NumberOfResourceGroupsToDelete int64
	ActuallyDelete                 bool
}

func (o Options) String() string {
	components := []string{
		fmt.Sprintf("Prefix %q", o.Prefix),
		fmt.Sprintf("Number RGs to Delete %d", o.NumberOfResourceGroupsToDelete),
		fmt.Sprintf("Actually Delete %t", o.ActuallyDelete),
	}
	return strings.Join(components, "\n")
}
