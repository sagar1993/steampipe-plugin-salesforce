package salesforce

import (
	"context"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func SalesforceOpportunity(ctx context.Context, dm dynamicMap, config salesforceConfig) *plugin.Table {
	tableName := "Opportunity"

	columns := mergeTableColumns(ctx, config, dm.cols, []*plugin.Column{
		// Top columns
		{Name: "id", Type: proto.ColumnType_STRING, Description: "Unique identifier of the opportunity in Salesforce."},
		{Name: "account_id", Type: proto.ColumnType_STRING, Description: "ID of the account associated with this opportunity."},
		{Name: "amount", Type: proto.ColumnType_DOUBLE, Description: "Estimated total sale amount. For opportunities with products, the amount is the sum of the related products."},
		{Name: "name", Type: proto.ColumnType_STRING, Description: "A name for this opportunity."},
		{Name: "owner_id", Type: proto.ColumnType_STRING, Description: "ID of the User who has been assigned to work this opportunity."},

		// Other columns
		{Name: "campaign_id", Type: proto.ColumnType_STRING, Description: "ID of a related Campaign. This field is defined only for those organizations that have the campaign feature Campaigns enabled."},
		{Name: "close_date", Type: proto.ColumnType_TIMESTAMP, Description: "Date when the opportunity is expected to close."},
		{Name: "created_by_id", Type: proto.ColumnType_STRING, Description: "Id of the user who created the opportunity."},
		{Name: "created_date", Type: proto.ColumnType_TIMESTAMP, Description: "The creation date and time of the opportunity."},
		{Name: "description", Type: proto.ColumnType_STRING, Description: "Description of the opportunity."},
		{Name: "expected_revenue", Type: proto.ColumnType_DOUBLE, Description: "Calculated revenue based on the Amount and Probability fields."},
		{Name: "fiscal_quarter", Type: proto.ColumnType_INT, Description: "Represents the fiscal quarter. Valid values are 1, 2, 3, or 4."},
		{Name: "fiscal_year", Type: proto.ColumnType_INT, Description: "Represents the fiscal year, for example, 2006."},
		{Name: "forecast_category", Type: proto.ColumnType_STRING, Description: "Forecast category name displayed in reports, opportunity detail and edit pages, opportunity searches, and opportunity list views."},
		{Name: "forecast_category_name", Type: proto.ColumnType_STRING, Description: "Name of the forecast category."},
		{Name: "has_open_activity", Type: proto.ColumnType_BOOL, Description: "Indicates whether an opportunity has an open event or task (true) or not (false)."},
		{Name: "has_opportunity_line_item", Type: proto.ColumnType_BOOL, Description: "Indicates whether the opportunity has associated line items. A value of true means that Opportunity line items have been created for the opportunity."},
		{Name: "has_overdue_task", Type: proto.ColumnType_BOOL, Description: "Indicates whether an opportunity has an overdue task (true) or not (false)."},
		{Name: "is_closed", Type: proto.ColumnType_BOOL, Description: "Indicates that the opportunity is closed."},
		{Name: "is_deleted", Type: proto.ColumnType_BOOL, Description: "Indicates that the opportunity is deleted."},
		{Name: "is_private", Type: proto.ColumnType_BOOL, Description: "Indicates that the opportunity is private."},
		{Name: "is_won", Type: proto.ColumnType_BOOL, Description: "Indicates that the quote or proposal has been signed or electronically accepted."},
		{Name: "last_activity_date", Type: proto.ColumnType_TIMESTAMP, Description: "Value is one of the following, whichever is the most recent of a) Due date of the most recent event logged against the record or b) Due date of the most recently closed task associated with the record."},
		{Name: "last_modified_by_id", Type: proto.ColumnType_STRING, Description: "The id of the user who last modified the oppurtinity record."},
		{Name: "last_modified_date", Type: proto.ColumnType_TIMESTAMP, Description: "The data and time of the last modification of the oppurtinity record."},
		{Name: "lead_source", Type: proto.ColumnType_STRING, Description: "Source of this opportunity, such as Advertisement or Trade Show."},
		{Name: "next_step", Type: proto.ColumnType_STRING, Description: "Description of next task in closing opportunity."},
		{Name: "pricebook_2_id", Type: proto.ColumnType_STRING, Description: "ID of a related Pricebook2 object. The Pricebook2Id field indicates which Pricebook2 applies to this opportunity. The Pricebook2Id field is defined only for those organizations that have products enabled as a feature."},
		{Name: "probability", Type: proto.ColumnType_DOUBLE, Description: "Percentage of estimated confidence in closing the opportunity."},
		{Name: "stage_name", Type: proto.ColumnType_STRING, Description: "Current stage of opportunity."},
		{Name: "system_modstamp", Type: proto.ColumnType_STRING, Description: "The date and time when opportunity was last modified by a user or by an automated process."},
		{Name: "total_opportunity_quantity", Type: proto.ColumnType_STRING, Description: "Number of items included in this opportunity. Used in quantity-based forecasting."},
		{Name: "type", Type: proto.ColumnType_STRING, Description: "Type of opportunity, such as Existing Business or New Business."},

		// Dynamic column to be removed later
		{Name: "acv__c", Type: proto.ColumnType_DOUBLE, Description: "TBD ACV.", Transform: transform.FromP(getFieldFromSObjectMap, "ACV__c")},
		{Name: "closed_reason__c", Type: proto.ColumnType_STRING, Description: "Opportunity Disqualified Reason.", Transform: transform.FromP(getFieldFromSObjectMap, "Closed_Reason__c")},
		{Name: "opportunity_category__c", Type: proto.ColumnType_STRING, Description: "Opportunity Category.", Transform: transform.FromP(getFieldFromSObjectMap, "Opportunity_Category__c")},
		{Name: "opportunity_m1_date_first__c", Type: proto.ColumnType_TIMESTAMP, Description: "Opportunity M1 Date First.", Transform: transform.FromP(getFieldFromSObjectMap, "Opportunity_M1_Date_First__c")},
		{Name: "opportunity_new_customer_software_acv__c", Type: proto.ColumnType_DOUBLE, Description: "Opportunity New Customer Software ACV.", Transform: transform.FromP(getFieldFromSObjectMap, "Opportunity_New_Customer_Software_ACV__c")},
		{Name: "opportunity_new_expand_acv__c", Type: proto.ColumnType_DOUBLE, Description: "Opportunity New/Expand Software ACV.", Transform: transform.FromP(getFieldFromSObjectMap, "Opportunity_New_Expand_ACV__c")},
		{Name: "opportunity_owner_id__c", Type: proto.ColumnType_STRING, Description: "Opportunity Owner ID.", Transform: transform.FromP(getFieldFromSObjectMap, "Opportunity_Owner_ID__c")},
		{Name: "opportunity_partner_account__c", Type: proto.ColumnType_STRING, Description: "Opportunity Partner Account.", Transform: transform.FromP(getFieldFromSObjectMap, "Opportunity_Partner_Account__c")},
		{Name: "opportunity_region__c", Type: proto.ColumnType_STRING, Description: "Opportunity Region.", Transform: transform.FromP(getFieldFromSObjectMap, "Opportunity_Region__c")},
		{Name: "opportunity_source_l__c", Type: proto.ColumnType_STRING, Description: "Opportunity Source Detailed.", Transform: transform.FromP(getFieldFromSObjectMap, "Opportunity_Source_L__c")},
		{Name: "opportunity_source_l_manual__c", Type: proto.ColumnType_STRING, Description: "Opportunity Source Detailed Manual.", Transform: transform.FromP(getFieldFromSObjectMap, "Opportunity_Source_L_Manual__c")},
		{Name: "opportunity_stage_etl__c", Type: proto.ColumnType_STRING, Description: "Opportunity Stage ETL.", Transform: transform.FromP(getFieldFromSObjectMap, "Opportunity_Stage_ETL__c")},
		{Name: "opportunity_stage_status__c", Type: proto.ColumnType_STRING, Description: "Opportunity Stage Status.", Transform: transform.FromP(getFieldFromSObjectMap, "Opportunity_Stage_Status__c")},
		{Name: "opportunity_status__c", Type: proto.ColumnType_STRING, Description: "Opportunity Status.", Transform: transform.FromP(getFieldFromSObjectMap, "Opportunity_Status__c")},
		{Name: "product_amount__c", Type: proto.ColumnType_DOUBLE, Description: "Opportunity Product TCV.", Transform: transform.FromP(getFieldFromSObjectMap, "Product_Amount__c")},
		{Name: "rd_forecast_category__c", Type: proto.ColumnType_STRING, Description: "RD Forecast Category.", Transform: transform.FromP(getFieldFromSObjectMap, "RD_Forecast_Category__c")},
		{Name: "tcv__c", Type: proto.ColumnType_DOUBLE, Description: "TCV.", Transform: transform.FromP(getFieldFromSObjectMap, "TCV__c")},
		{Name: "tvp_forecast_category__c", Type: proto.ColumnType_STRING, Description: "TVP Forecast Category.", Transform: transform.FromP(getFieldFromSObjectMap, "TVP_Forecast_Category__c")},
	})

	plugin.Logger(ctx).Debug("SalesforceOpportunity init")

	queryColumnsMap := make(map[string]*plugin.Column)
	for _, column := range columns {
		queryColumnsMap[getSalesforceColumnName(column.Name)] = column
	}

	return &plugin.Table{
		Name:        "salesforce_opportunity",
		Description: "Represents an opportunity, which is a sale or pending deal.",
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
