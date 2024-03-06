package salesforce

import (
	"context"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
)

func SalesforceCase(ctx context.Context, dm dynamicMap, config salesforceConfig) *plugin.Table {
	tableName := "Case"

	columns := mergeTableColumns(ctx, config, dm.cols, []*plugin.Column{ //
		// Top columns
		{Name: "id", Type: proto.ColumnType_STRING, Description: "Unique identifier of the account in Salesforce."},
		{Name: "is_deleted", Type: proto.ColumnType_BOOL, Description: ""},
		{Name: "case_number", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "contact_id", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "account_id", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "asset_id", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "product_id", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "entitlement_id", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "source_id", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "business_hours_id", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "parent_id", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "supplied_name", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "supplied_email", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "supplied_phone", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "supplied_company", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "type", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "record_type_id", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "status", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "reason", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "origin", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "language", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "subject", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "priority", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "description", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "is_closed", Type: proto.ColumnType_BOOL, Description: ""},
		{Name: "closed_date", Type: proto.ColumnType_TIMESTAMP, Description: ""},
		{Name: "is_escalated", Type: proto.ColumnType_BOOL, Description: ""},
		{Name: "currency_iso_code", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "owner_id", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "is_closed_on_create", Type: proto.ColumnType_BOOL, Description: ""},
		{Name: "sla_start_date", Type: proto.ColumnType_TIMESTAMP, Description: ""},
		{Name: "sla_exit_date", Type: proto.ColumnType_TIMESTAMP, Description: ""},
		{Name: "is_stopped", Type: proto.ColumnType_BOOL, Description: ""},
		{Name: "stop_start_date", Type: proto.ColumnType_TIMESTAMP, Description: ""},
		{Name: "created_date", Type: proto.ColumnType_TIMESTAMP, Description: ""},
		{Name: "created_by_id", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "last_modified_date", Type: proto.ColumnType_TIMESTAMP, Description: ""},
		{Name: "last_modified_by_id", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "system_modstamp", Type: proto.ColumnType_TIMESTAMP, Description: ""},
		{Name: "contact_phone", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "contact_mobile", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "contact_email", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "contact_fax", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "comments", Type: proto.ColumnType_STRING, Description: ""},
		{Name: "last_viewed_date", Type: proto.ColumnType_TIMESTAMP, Description: ""},
		{Name: "last_referenced_date", Type: proto.ColumnType_TIMESTAMP, Description: ""},
		{Name: "milestone_status", Type: proto.ColumnType_STRING, Description: ""},
	})

	plugin.Logger(ctx).Debug("SalesforceCase init")

	queryColumnsMap := make(map[string]*plugin.Column)
	for _, column := range columns {
		queryColumnsMap[getSalesforceColumnName(column.Name)] = column
	}

	return &plugin.Table{
		Name:        "salesforce_case",
		Description: "Represents case records.",
		List: &plugin.ListConfig{
			Hydrate:    listSalesforceObjectsByTable(tableName, dm.salesforceColumns, queryColumnsMap),
			KeyColumns: getKeyColumns(columns),
		},
		Get: &plugin.GetConfig{
			Hydrate:    getSalesforceObjectbyID(tableName, queryColumnsMap),
			KeyColumns: plugin.SingleColumn(checkNameScheme(config, dm.cols)),
		},
		Columns: columns,
	}
}
