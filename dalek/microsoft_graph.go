package dalek

import (
	"context"
	"fmt"
	"log"

	"github.com/tombuildsstuff/azurerm-dalek/dalek/cleaners"
)

func (d *Dalek) MicrosoftGraph(ctx context.Context) error {
	for _, cleaner := range cleaners.MsGraphCleaners {
		log.Printf("[DEBUG] Running Microsoft Graph Cleaner %q", cleaner.Name())
		if err := cleaner.Cleanup(ctx, d.client, d.opts); err != nil {
			return fmt.Errorf("running Microsoft Graph Cleaner %q: %+v", cleaner.Name(), err)
		}
	}

	return nil
}
