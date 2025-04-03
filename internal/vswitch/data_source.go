package vswitch

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

// DataSourceType is the type name of the Hetzner Robot vSwitch resource.
const DataSourceType = "hetznerrobot_vswitch"

func DataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceRead,
		Schema: map[string]*schema.Schema{
			"ids": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"vswitches": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":        {Type: schema.TypeString, Computed: true},
						"name":      {Type: schema.TypeString, Computed: true},
						"vlan":      {Type: schema.TypeInt, Computed: true},
						"cancelled": {Type: schema.TypeBool, Computed: true},
					},
				},
			},
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	idsInterface := d.Get("ids").([]any)
	ids := make([]string, 0, len(idsInterface))
	for _, id := range idsInterface {
		ids = append(ids, id.(string))
	}

	var (
		vswitches []client.VSwitch
		err       error
	)

	if len(ids) == 0 {
		vswitches, err = hClient.FetchAllVSwitches(ctx)
		if err != nil {
			return diag.FromErr(fmt.Errorf("error fetching ALL vSwitches: %w", err))
		}
	} else {
		vswitches, err = hClient.FetchVSwitchesByIDs(ctx, ids)
		if err != nil {
			return diag.FromErr(fmt.Errorf("error fetching vSwitches by IDs: %w", err))
		}
	}

	if len(vswitches) == 0 {
		return diag.FromErr(fmt.Errorf("no vSwitches found"))
	}

	if err := d.Set("vswitches", flattenVSwitches(vswitches)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting vswitches attribute: %w", err))
	}

	idStr := "all"
	if len(ids) > 0 {
		idStr = strings.Join(ids, "-")
	}

	d.SetId(fmt.Sprintf("vswitches-%s", idStr))

	return nil
}

func flattenVSwitches(vswitches []client.VSwitch) []map[string]any {
	res := make([]map[string]any, 0, len(vswitches))
	for _, vs := range vswitches {
		res = append(res, map[string]any{
			"id":        strconv.Itoa(vs.ID),
			"name":      vs.Name,
			"vlan":      vs.VLAN,
			"cancelled": vs.Cancelled,
		})
	}

	return res
}
