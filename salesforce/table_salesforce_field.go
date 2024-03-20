package salesforce

import (
	"context"
	"strings"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func SalesforceField(ctx context.Context, dc map[string]dynamicMap, config salesforceConfig) *plugin.Table {
	// tableName := "Field"

	plugin.Logger(ctx).Debug("Field init")

	return &plugin.Table{
		Name:        "salesforce_field",
		Description: "A custom field in Salesforce.",
		List: &plugin.ListConfig{
			Hydrate: listFields(dc),
			KeyColumns: plugin.KeyColumnSlice{
				{Name: "name", Require: plugin.Optional, Operators: []string{"=", "<>"}},
				{Name: "type", Require: plugin.Optional, Operators: []string{"=", "<>"}},
			},
		},
		Columns: []*plugin.Column{
			{
				Name:        "name",
				Description: "Name of the Salesforce custom field.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("name"),
			},
			{
				Name:        "type",
				Description: "Data type of the Salesforce custom field.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("type"),
			},
		},
	}
}

func listFields(dc map[string]dynamicMap) func(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	return func(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
		for _, m := range dc {
			for colName, colType := range m.salesforceColumns {
				if strings.HasSuffix(colName, "__c") {
					row := map[string]interface{}{
						"name": colName,
						"type": colType,
					}
					d.StreamListItem(ctx, row)

					// Context may get cancelled due to manual cancellation or if the limit has been reached
					if d.RowsRemaining(ctx) == 0 {
						return nil, nil
					}
				}
			}
		}
		return nil, nil
	}
}
