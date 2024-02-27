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
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
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

var cacheExpiration = 10 * time.Minute
var cleanupInterval = 10 * time.Minute
var batchSize = 500
var idFormatter = func(id string) string {
	return fmt.Sprintf("'%s'", id)
}

var cacheUtil = cache.NewCacheUtil(tableKeyStruct, cacheExpiration, cleanupInterval, batchSize, idFormatter)

func listSalesforceObjectsByTable(tableName string, salesforceCols map[string]string, queryColumnsMap map[string]*plugin.Column) func(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	return func(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
		startTime := time.Now()
		defer measureTime(ctx, startTime, "listSalesforceObjectsByTable")
		client, err := connect(ctx, d)
		if err != nil {
			plugin.Logger(ctx).Error("salesforce.listSalesforceObjectsByTable", "connection error", err)
			return nil, err
		}
		if client == nil {
			plugin.Logger(ctx).Error("salesforce.listSalesforceObjectsByTable", "client_not_found: unable to generate dynamic tables because of invalid steampipe salesforce configuration", err)
			return nil, fmt.Errorf("salesforce.listSalesforceObjectsByTable: client_not_found, unable to query table %s because of invalid steampipe salesforce configuration", d.Table.Name)
		}

		var queryColumns []*plugin.Column
		for _, element := range d.QueryContext.Columns {
			if column, ok := queryColumnsMap[getSalesforceColumnName(element)]; ok {
				queryColumns = append(queryColumns, column)
			} else {
				plugin.Logger(ctx).Error("salesforce.listSalesforceObjectsByTable", "no column found", queryColumnsMap, element)
			}
		}

		query := generateQuery(queryColumns, tableName)
		condition := buildQueryFromQuals(d.Quals, queryColumns, salesforceCols)
		if condition != "" {
			query = fmt.Sprintf("%s where %s", query, condition)
			plugin.Logger(ctx).Debug("salesforce.listSalesforceObjectsByTable", "table_name", d.Table.Name, "query_condition", condition)
		}
		salesforceConfig := GetConfig(d.Connection)

		if isColumnAvailable("last_modified_date", d.Table.Columns) {
			query = fmt.Sprintf("%s  order by lastModifiedDate desc", query)
		}

		configLimit := math.MaxInt32
		if salesforceConfig.ResultSize != nil {
			configLimit = *salesforceConfig.ResultSize + 1
		}
		if d.QueryContext.Limit != nil && int(*d.QueryContext.Limit) < configLimit {
			configLimit = int(*d.QueryContext.Limit)
		}
		if 0 < configLimit && configLimit < math.MaxInt32 {
			limitString := strconv.Itoa(configLimit)
			query = fmt.Sprintf("%s  limit %s", query, limitString)
		}

		var totalRecords int = 0
		var dataList [][]map[string]interface{}
		for {
			plugin.Logger(ctx).Debug("salesforce.listSalesforceObjectsByTable getting results for query : ", query)

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

			totalRecords += len(*AccountList)
			if *salesforceConfig.ShowResultSizeError && totalRecords > *salesforceConfig.ResultSize {
				return nil, fmt.Errorf("Query returned too many rows, please add a few filters to reduce it.")
			}

			dataList = append(dataList, *AccountList)

			// Paging
			if result.Done {
				break
			} else {
				query = result.NextRecordsURL
			}
		}
		for _, data := range dataList {
			for _, account := range data {
				for col_name, col_value := range account {
					if column_obj, ok := queryColumnsMap[col_name]; ok {
						if column_obj.Type == proto.ColumnType_STRING && col_value != nil && !strings.HasSuffix(col_name, "id") && !strings.HasSuffix(col_name, "Id") {
							account[col_name] = strings.ToLower(col_value.(string))
						}
					}
				}
				cacheUtil.AddIdsToForeignTableCache(ctx, getTableName(tableName), account)
			}

			for _, account := range data {
				d.StreamListItem(ctx, account)
			}
		}

		return nil, nil
	}
}

func bulkDataPullByIds(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData, ids []string, columnsMap map[string]*plugin.Column) (*[]map[string]interface{}, error) {
	startTime := time.Now()
	defer measureTime(ctx, startTime, "bulkDataPullByIds")

	var queryColumns []*plugin.Column
	for _, element := range columnsMap {
		queryColumns = append(queryColumns, element)
	}
	// make query call to get data and update cache
	// make query call to get data
	query := generateQuery(queryColumns, getTableName(d.Table.Name))

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

	data := new([]map[string]interface{})
	for {
		result, err := client.Query(query)
		if err != nil {
			plugin.Logger(ctx).Error("salesforce.bulkDataPullByIds", "query error", err)
			return nil, err
		}
		temp := new([]map[string]interface{})
		err = decodeQueryResult(ctx, result.Records, temp)
		if err != nil {
			plugin.Logger(ctx).Error("salesforce.bulkDataPullByIds", "results decoding error", err)
			return nil, err
		}
		for _, data := range *temp {
			for col_name, col_value := range data {
				if column_obj, ok := columnsMap[col_name]; ok {
					if column_obj.Type == proto.ColumnType_STRING && col_value != nil && !strings.HasSuffix(col_name, "id") && !strings.HasSuffix(col_name, "Id") {
						data[col_name] = strings.ToLower(col_value.(string))
					}
				}
			}
		}
		// Paging
		if result.Done {
			*data = append(*data, *temp...)
			break
		} else {
			query = result.NextRecordsURL
		}

	}
	return data, nil
}

func getSalesforceObjectbyID(tableName string, queryColumnsMap map[string]*plugin.Column) func(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	return func(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
		startTime := time.Now()
		defer measureTime(ctx, startTime, "getSalesforceObjectbyID")
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

		columnsMap := make(map[string]*plugin.Column)
		for _, name := range d.QueryContext.Columns {
			if column, ok := queryColumnsMap[getSalesforceColumnName(name)]; ok {
				columnsMap[getSalesforceColumnName(name)] = column
			}
		}

		record, err := cacheUtil.GetRecordByIdAndBuildCache(ctx, d, h, getTableName(tableName), id, columnsMap)
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

func measureTime(ctx context.Context, start time.Time, functionName string) {
	plugin.Logger(ctx).Debug(fmt.Sprintf("Function %s took %s\n", functionName, time.Since(start)))
}
