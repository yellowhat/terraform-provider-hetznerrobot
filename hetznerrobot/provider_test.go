package hetznerrobot_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yellowhat/terraform-provider-hetznerrobot/hetznerrobot"
	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/firewall"
	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/server"
	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/vswitch"
)

func TestProvider(t *testing.T) {
	t.Parallel()

	if err := hetznerrobot.Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_Resources(t *testing.T) {
	t.Parallel()

	provider := hetznerrobot.Provider()
	expectedResources := []string{
		firewall.ResourceType,
		server.ResourceOSRescueType,
		vswitch.ResourceType,
	}

	resources := provider.Resources()
	assert.Len(t, resources, len(expectedResources))

	for _, resource := range resources {
		assert.Contains(t, expectedResources, resource.Name)
	}
}

func TestProvider_DataSources(t *testing.T) {
	t.Parallel()

	provider := hetznerrobot.Provider()
	expectedDataSources := []string{
		server.DataSourceType,
		vswitch.DataSourceType,
	}

	dataSources := provider.DataSources()
	assert.Len(t, dataSources, len(expectedDataSources))

	for _, datasource := range dataSources {
		assert.Contains(t, expectedDataSources, datasource.Name)
	}
}
