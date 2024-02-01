package salesforce

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/turbot/steampipe-plugin-salesforce/cache"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

//// LIST HYDRATE FUNCTION

var tableKeyStruct = []cache.KeyStruct{
	{
		Name:              "Account",
		Pk:                "Id",
		Fk:                []cache.ForeignKeyStruct{},
		BulkDataPullByIds: bulkDataPullByIds,
	}, {
		Name: "Opportunity",
		Pk:   "Id",
		Fk: []cache.ForeignKeyStruct{{
			Key:              "AccountId",
			ForeignTableName: "Account",
		}},
		BulkDataPullByIds: bulkDataPullByIds,
	}, {
		Name: "Case",
		Pk:   "Id",
		Fk: []cache.ForeignKeyStruct{{
			Key:              "AccountId",
			ForeignTableName: "Account",
		}},
		BulkDataPullByIds: bulkDataPullByIds,
	}, {
		Name: "Order",
		Pk:   "Id",
		Fk: []cache.ForeignKeyStruct{{
			Key:              "AccountId",
			ForeignTableName: "Account",
		}},
		BulkDataPullByIds: bulkDataPullByIds,
	},
}

var cacheExpiration = 1 * time.Minute
var cleanupInterval = 2 * time.Minute
var batchSize = 200
var idFormatter = func(id string) string {
	return fmt.Sprintf("'%s'", id)
}

var cacheUtil = cache.NewCacheUtil(tableKeyStruct, cacheExpiration, cleanupInterval, batchSize, idFormatter)

func listSalesforceObjectsByTable(tableName string, salesforceCols map[string]string) func(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	return func(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
		client, err := connect(ctx, d)
		if err != nil {
			plugin.Logger(ctx).Error("salesforce.listSalesforceObjectsByTable", "connection error", err)
			return nil, err
		}
		if client == nil {
			plugin.Logger(ctx).Error("salesforce.listSalesforceObjectsByTable", "client_not_found: unable to generate dynamic tables because of invalid steampipe salesforce configuration", err)
			return nil, fmt.Errorf("salesforce.listSalesforceObjectsByTable: client_not_found, unable to query table %s because of invalid steampipe salesforce configuration", d.Table.Name)
		}

		query := generateQuery(d.Table.Columns, tableName)
		condition := buildQueryFromQuals(d.Quals, d.Table.Columns, salesforceCols)
		if condition != "" {
			query = fmt.Sprintf("%s where %s", query, condition)
			plugin.Logger(ctx).Debug("salesforce.listSalesforceObjectsByTable", "table_name", d.Table.Name, "query_condition", condition)
		}
		salesforceConfig := GetConfig(d.Connection)

		if isColumnAvailable("last_modified_date", d.Table.Columns) {
			query = fmt.Sprintf("%s  order by lastModifiedDate desc", query)
		}

		if salesforceConfig.ResultSize != nil {
			intValue := *salesforceConfig.ResultSize
			if 0 < intValue && intValue < math.MaxInt32 {
				limitString := strconv.Itoa(intValue)
				query = fmt.Sprintf("%s  limit %s", query, limitString)
			}
		}
		plugin.Logger(ctx).Debug("## getting results for query : ", query)

		for {
			result, err := client.Query(query)
			if err != nil {
				plugin.Logger(ctx).Error("salesforce.listSalesforceObjectsByTable", "query error", err)
				return nil, err
			}

			AccountList := new([]map[string]interface{})
			err = decodeQueryResult(ctx, result.Records, AccountList)
			if err != nil {
				plugin.Logger(ctx).Error("salesforce.listSalesforceObjectsByTable", "results decoding error", err)
				return nil, err
			}

			for _, account := range *AccountList {
				cacheUtil.AddIdsToForeignTableCache(getTableName(tableName), account)
				d.StreamListItem(ctx, account)
			}

			// Paging
			if result.Done {
				break
			} else {
				query = result.NextRecordsURL
			}
		}

		return nil, nil
	}
}

func bulkDataPullByIds(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData, ids []string) (*[]map[string]interface{}, error) {
	// make query call to get data and update cache
	// make query call to get data
	query := generateQuery(d.Table.Columns, getTableName(d.Table.Name))

	// Concatenate the values into a comma-separated string
	inClause := strings.Join(ids, ",")

	// Create the WHERE clause
	whereClause := fmt.Sprintf("WHERE Id IN (%s)", inClause)
	query = fmt.Sprintf("%s  %s", query, whereClause)

	client, err := connect(ctx, d)
	if err != nil {
		plugin.Logger(ctx).Error("salesforce.bulkDataPullByIds", "connection error", err)
		return nil, err
	}
	if client == nil {
		plugin.Logger(ctx).Error("salesforce.bulkDataPullByIds", "client_not_found: unable to generate dynamic tables because of invalid steampipe salesforce configuration", err)
		return nil, fmt.Errorf("salesforce.bulkDataPullByIds: client_not_found, unable to query table %s because of invalid steampipe salesforce configuration", d.Table.Name)
	}

	plugin.Logger(ctx).Debug("salesforce.bulkDataPullByIds GET getting results for query : ", query)

	result, err := client.Query(query)
	if err != nil {
		plugin.Logger(ctx).Error("salesforce.bulkDataPullByIds", "query error", err)
		return nil, err
	}

	data := new([]map[string]interface{})
	err = decodeQueryResult(ctx, result.Records, data)
	if err != nil {
		plugin.Logger(ctx).Error("salesforce.bulkDataPullByIds", "results decoding error", err)
		return nil, err
	}
	return data, nil
}

func getSalesforceObjectbyID(tableName string) func(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	return func(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
		plugin.Logger(ctx).Info("salesforce.getSalesforceObjectbyID", "Table_Name", d.Table.Name)
		config := GetConfig(d.Connection)
		var id string
		if config.NamingConvention != nil && *config.NamingConvention == "api_native" {
			id = d.EqualsQualString("Id")
		} else {
			id = d.EqualsQualString("id")
		}
		if strings.TrimSpace(id) == "" {
			return nil, nil
		}

		record, err := cacheUtil.GetRecordByIdAndBuildCache(ctx, d, h, getTableName(tableName), id)
		if record != nil {
			return record, nil
		}
		if err != nil {
			plugin.Logger(ctx).Error("salesforce.getSalesforceObjectbyID", "error getting record from cache", err)
		}

		client, err := connect(ctx, d)
		if err != nil {
			plugin.Logger(ctx).Error("salesforce.getSalesforceObjectbyID", "connection error", err)
			return nil, err
		}
		if client == nil {
			plugin.Logger(ctx).Error("salesforce.getSalesforceObjectbyID", "client_not_found: unable to generate dynamic tables because of invalid steampipe salesforce configuration", err)
			return nil, fmt.Errorf("salesforce.getSalesforceObjectbyID: client_not_found, unable to query table %s because of invalid steampipe salesforce configuration", d.Table.Name)
		}

		obj := client.SObject(tableName).Get(id)
		if obj == nil {
			// Object doesn't exist, handle the error
			plugin.Logger(ctx).Error("salesforce.getSalesforceObjectbyID", fmt.Sprintf("%s with id \"%s\" not found", tableName, id))
			return nil, nil
		}

		object := new(map[string]interface{})
		err = decodeQueryResult(ctx, obj, object)
		if err != nil {
			plugin.Logger(ctx).Error("salesforce.getSalesforceObjectbyID", "result decoding error", err)
			return nil, err
		}

		return *object, nil
	}
}

//// TRANSFORM FUNCTION

func getFieldFromSObjectMap(ctx context.Context, d *transform.TransformData) (interface{}, error) {
	param := d.Param.(string)
	ls := d.HydrateItem.(map[string]interface{})
	return ls[param], nil
}

func getFieldFromSObjectMapByColumnName(ctx context.Context, d *transform.TransformData) (interface{}, error) {
	salesforceColumnName := getSalesforceColumnName(d.ColumnName)
	ls := d.HydrateItem.(map[string]interface{})
	return ls[salesforceColumnName], nil
}

// convert tablename salesforce_abc to Abc
func getTableName(input string) string {
	// Check if the string starts with "salesforce_"
	if strings.HasPrefix(input, "salesforce_") {
		// Remove "salesforce_"
		trimmed := strings.TrimPrefix(input, "salesforce_")
		// Convert to camelCase
		camelCase := strcase.ToCamel(trimmed)
		return camelCase
	}
	// If the string doesn't start with "salesforce_", return it as is
	return input
}
