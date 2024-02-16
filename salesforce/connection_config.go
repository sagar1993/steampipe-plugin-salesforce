package salesforce

import (
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/schema"
)

type NamingConventionEnum string

const (
	API_NATIVE NamingConventionEnum = "api_native"
	SNAKE_CASE NamingConventionEnum = "snake_case"
)

type salesforceConfig struct {
	URL                            *string               `cty:"url"`
	Username                       *string               `cty:"username"`
	Password                       *string               `cty:"password"`
	Token                          *string               `cty:"token"`
	ClientId                       *string               `cty:"client_id"`
	APIVersion                     *string               `cty:"api_version"`
	Objects                        *[]string             `cty:"objects"`
	NamingConvention               *NamingConventionEnum `cty:"naming_convention"`
	UserDefinedDynamicColumnConfig *string               `cty:"user_defined_dynamic_column_config"`
	ResultSize                     *int                  `cty:"result_size"`
	ShowResultSizeError            *bool                 `cty:"show_result_size_error"`
	CaseSenistiveSearch            *bool                 `cty:"case_sensitive_search"`
}

type UserDefinedDynamicColumnConfig struct {
	Name    string   `json:name`
	Columns []string `json:columns`
}

var ConfigSchema = map[string]*schema.Attribute{
	"url": {
		Type: schema.TypeString,
	},
	"naming_convention": {
		Type: schema.TypeString,
	},
	"username": {
		Type: schema.TypeString,
	},
	"password": {
		Type: schema.TypeString,
	},
	"token": {
		Type: schema.TypeString,
	},
	"client_id": {
		Type: schema.TypeString,
	},
	"api_version": {
		Type: schema.TypeString,
	},
	"objects": {
		Type: schema.TypeList,
		Elem: &schema.Attribute{
			Type: schema.TypeString,
		},
	},
	"user_defined_dynamic_column_config": {
		Type: schema.TypeString,
	},
	"result_size": {
		Type: schema.TypeInt,
	},
	"show_result_size_error": {
		Type: schema.TypeBool,
	},
}

func ConfigInstance() interface{} {
	return &salesforceConfig{}
}

// GetConfig :: retrieve and cast connection config from query data
func GetConfig(connection *plugin.Connection) salesforceConfig {
	if connection == nil || connection.Config == nil {
		return salesforceConfig{}
	}
	defaultShowResultSize := true
	config, _ := connection.Config.(salesforceConfig)
	if config.ShowResultSizeError == nil {
		config.ShowResultSizeError = &defaultShowResultSize
	}
	defaultCaseSenistiveSearch := true
	if config.ShowResultSizeError == nil {
		config.CaseSenistiveSearch = &defaultCaseSenistiveSearch
	}
	return config
}
