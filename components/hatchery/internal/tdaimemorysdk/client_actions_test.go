package tdaimemorysdk

import (
	"context"
	"testing"
)

func TestWrapperMethods_NilClientReturnsError(t *testing.T) {
	var c *Client
	calls := []struct {
		name string
		fn   func() error
	}{
		{"DescribeAgentInstance", func() error { _, err := c.DescribeAgentInstance(context.Background(), nil); return err }},
		{"CreateMemoryProInstance", func() error { _, err := c.CreateMemoryProInstance(context.Background(), nil); return err }},
		{"DescribeMemoryProInstances", func() error { _, err := c.DescribeMemoryProInstances(context.Background(), nil); return err }},
		{"ModifyMemoryProInstance", func() error { _, err := c.ModifyMemoryProInstance(context.Background(), nil); return err }},
		{"DeleteMemoryProInstance", func() error { _, err := c.DeleteMemoryProInstance(context.Background(), nil); return err }},
		{"CreateMemSpace", func() error { _, err := c.CreateMemSpace(context.Background(), nil); return err }},
		{"DescribeMemSpaces", func() error { _, err := c.DescribeMemSpaces(context.Background(), nil); return err }},
		{"DescribeMemSpaceRecord", func() error { _, err := c.DescribeMemSpaceRecord(context.Background(), nil); return err }},
		{"DeleteMemSpace", func() error { _, err := c.DeleteMemSpace(context.Background(), nil); return err }},
	}

	for _, tc := range calls {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err == nil {
				t.Fatal("expected error from nil client")
			}
		})
	}
}
