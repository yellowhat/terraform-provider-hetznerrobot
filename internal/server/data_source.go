// Package server defines the server terraform datasource and resource.
package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

// DataSourceType is the type name of the Hetzner Robot Server resource.
const DataSourceType = "hetznerrobot_server"

// DataSourceServers defines the servers terraform datasource.
func DataSourceServers() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceServersRead,
		Schema: map[string]*schema.Schema{
			"ids": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"servers": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip":         {Type: schema.TypeString, Computed: true},
						"ipv6_net":   {Type: schema.TypeString, Computed: true},
						"number":     {Type: schema.TypeInt, Computed: true},
						"name":       {Type: schema.TypeString, Computed: true},
						"product":    {Type: schema.TypeString, Computed: true},
						"datacenter": {Type: schema.TypeString, Computed: true},
						"traffic":    {Type: schema.TypeString, Computed: true},
						"status":     {Type: schema.TypeString, Computed: true},
						"cancelled":  {Type: schema.TypeBool, Computed: true},
						"paid_until": {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceServersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	rawIDs := d.Get("ids").([]any)
	ids := make([]string, 0, len(rawIDs))

	for _, v := range rawIDs {
		ids = append(ids, v.(string))
	}

	var (
		servers []client.Server
		err     error
	)

	if len(ids) == 0 {
		servers, err = hClient.FetchAllServers(ctx)
	} else {
		servers, err = hClient.FetchServersByIDs(ctx, ids)
	}

	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to fetch servers: %w", err))
	}

	serverList := make([]map[string]any, 0, len(servers))
	for _, s := range servers {
		serverList = append(serverList, map[string]any{
			"ip":         s.IP,
			"ipv6_net":   s.IPv6Net,
			"number":     s.Number,
			"name":       s.ServerName,
			"product":    s.Product,
			"datacenter": s.Datacenter,
			"traffic":    s.Traffic,
			"status":     s.Status,
			"cancelled":  s.Cancelled,
			"paid_until": s.PaidUntil,
		})
	}

	err = d.Set("servers", serverList)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error setting servers attribute: %w", err))
	}

	idStr := "all"
	if len(ids) > 0 {
		idStr = strings.Join(ids, "-")
	}

	d.SetId("servers-" + idStr)

	return nil
}
