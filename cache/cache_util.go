package cache

import (
	"context"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
)

type CacheUtil struct {
	tableCache        map[string]*cache.Cache
	tableIdSet        map[string]*Set
	tableKeyStructMap map[string]*KeyStruct
	IdFormatter       func(string) string
	TableKeyStruct    []KeyStruct
	cacheExpiration   time.Duration
	cleanupInterval   time.Duration
	batchSize         int
}

func generateTableCache(tableKeyStruct []KeyStruct, cacheExpiration time.Duration, cleanupInterval time.Duration) map[string]*cache.Cache {
	tableCache := make(map[string]*cache.Cache)
	for _, keyStruct := range tableKeyStruct {
		tableCache[keyStruct.Name] = cache.New(cacheExpiration, cleanupInterval)
	}
	return tableCache
}

func generateIdSet(tableKeyStruct []KeyStruct) map[string]*Set {
	tableIdSet := make(map[string]*Set)
	for _, keyStruct := range tableKeyStruct {
		tableIdSet[keyStruct.Name] = NewSet()
	}
	return tableIdSet
}

func generateTableKeyStructMap(tableKeyStruct []KeyStruct) map[string]*KeyStruct {
	tableKeyStructMap := make(map[string]*KeyStruct)

	for _, keyStruct := range tableKeyStruct {
		tableKeyStructMap[keyStruct.Name] = &keyStruct
	}
	return tableKeyStructMap
}

func NewCacheUtil(tableKeyStruct []KeyStruct, cacheExpiration time.Duration, cleanupInterval time.Duration, batchSize int, idFormatter func(id string) string) *CacheUtil {

	// if batch size is not provided, set it to 50
	if batchSize <= 0 {
		batchSize = 50
	}

	if idFormatter == nil {
		idFormatter = func(id string) string {
			return id
		}
	}

	return &CacheUtil{
		tableCache:        generateTableCache(tableKeyStruct, cacheExpiration, cleanupInterval),
		tableIdSet:        generateIdSet(tableKeyStruct),
		TableKeyStruct:    tableKeyStruct,
		tableKeyStructMap: generateTableKeyStructMap(tableKeyStruct),
		IdFormatter:       idFormatter,
		cacheExpiration:   cacheExpiration,
		cleanupInterval:   cleanupInterval,
		batchSize:         batchSize,
	}
}

func (c *CacheUtil) getIdSetForTableName(tableName string) *Set {
	if idSet, ok := c.tableIdSet[tableName]; ok {
		return idSet
	}
	return nil
}

func (c *CacheUtil) getCacheForTableName(tableName string) *cache.Cache {
	if cache, ok := c.tableCache[tableName]; ok {
		return cache
	}
	return nil
}

func (c *CacheUtil) getKeyStructForTableName(tableName string) *KeyStruct {
	if keyStruct, ok := c.tableKeyStructMap[tableName]; ok {
		return keyStruct
	}
	return nil
}

// TODO create generator function for this
// so that memory can be freed up
// discards any entries in cache.
func (c *CacheUtil) getKeysToPullInBatches(tableName string, batchSize int) [][]string {
	var result [][]string
	var currentTime = time.Now()
	var currentBatch []string

	cache := c.getCacheForTableName(tableName)

	var idSet *Set
	idSet = c.getIdSetForTableName(tableName)

	for key, time := range idSet.set {
		if cache != nil {
			if record, exists := cache.Get(key); exists {
				// if required element is already in cache increment TTL for the element in cache
				cache.Set(key, record, c.cacheExpiration)
				continue
			}
		}

		currentBatch = append(currentBatch, c.IdFormatter(key))

		if len(currentBatch) == batchSize {
			result = append(result, currentBatch)
			currentBatch = []string{}
		}
		if time.After(currentTime) {
			break
		}
	}
	if len(currentBatch) > 0 {
		result = append(result, currentBatch)
	}
	return result
}

// The function is used along with the List call in plugin and adds the ids to the id cache of the foreign table
// and records to the table cache
func (c *CacheUtil) AddIdsToForeignTableCache(tableName string, record map[string]interface{}) {
	keyStruct := c.getKeyStructForTableName(tableName)
	// Add foreign keys to the id set
	for _, fk := range keyStruct.Fk {
		id, exists := record[fk.Key]
		if exists {
			if idValue, ok := id.(string); ok {
				c.getIdSetForTableName(fk.ForeignTableName).Add(idValue)
			}
		}
	}

	// add record to the table cache
	id, exists := record[keyStruct.Pk]
	if exists {
		if idValue, ok := id.(string); ok {
			c.getCacheForTableName(tableName).Set(idValue, record, c.cacheExpiration)
		}
	}
}

// The function is used along with the Get call in plugin, it returns the record from the cache if it exists
// otherwise it pulls the records from the data source and adds it to the cache
func (c *CacheUtil) GetRecordByIdAndBuildCache(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData, tableName string, idToReturn string) (interface{}, error) {
	var tableCache = c.getCacheForTableName(tableName)
	var idSet = c.getIdSetForTableName(tableName)
	var keyStruct = c.getKeyStructForTableName(tableName)

	//--------------- Getting values from the cache ------------------//

	if value, exists := tableCache.Get(idToReturn); exists {
		return value, nil
	} else {
		plugin.Logger(ctx).Debug("salesforce.GetRecordByIdAndBuildCache ID not present in cache 1st check ", idToReturn)
	}

	//--------------- Build cache in batches ------------------//
	var batches = c.getKeysToPullInBatches(tableName, c.batchSize)

	for _, batch := range batches {

		DataList, err := keyStruct.BulkDataPullByIds(ctx, d, h, batch)
		if err != nil {
			plugin.Logger(ctx).Debug("salesforce.GetRecordByIdAndBuildCache", "results decoding error", err)
		}

		for _, record := range *DataList {
			// Accessing a specific key
			id, exists := record[keyStruct.Pk]
			if exists {
				// Convert the interface{} to a string using type assertion
				if idValue, ok := id.(string); ok {
					// Setting the value in the cache
					tableCache.Set(idValue, record, c.cacheExpiration)
					// Removing the id from the set
					idSet.Remove(idValue)

					c.AddIdsToForeignTableCache(tableName, record)

				} else {
					plugin.Logger(ctx).Debug("salesforce.GetRecordByIdAndBuildCache cache set failed ", id, " id value ", idValue)
				}
			} else {
				plugin.Logger(ctx).Debug("salesforce.GetRecordByIdAndBuildCache cache set idString does not exists", id, " value ", record)
			}
		}
	}

	//--------------- Getting values from the cache built------------------//

	if value, exists := tableCache.Get(idToReturn); exists {
		return value, nil
	} else {
		plugin.Logger(ctx).Debug("salesforce.GetRecordByIdAndBuildCache not present in cache ", idToReturn)
	}

	return nil, nil
}

//--------------- SET  ------------------//

// Replace with time based map with TTL

type Set struct {
	mu  sync.Mutex
	set map[string]time.Time
}

// Add adds an element to the set
func (s *Set) Add(entry string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.set[entry] = time.Now()
}

// Remove removes an element from the set
func (s *Set) Remove(entry string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.set, entry)
}

// Contains checks if the set contains a specific element
func (s *Set) Contains(entry string) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set[entry]
}

func NewSet() *Set {
	return &Set{
		set: make(map[string]time.Time),
	}
}

//--------------- SET END ------------------//

//--------------- KeyStruct ------------------//

type BulkDataPullByIdsFunc func(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData, ids []string) (*[]map[string]interface{}, error)

type KeyStruct struct {
	Name              string
	Pk                string
	Fk                []ForeignKeyStruct
	BulkDataPullByIds BulkDataPullByIdsFunc
}

type ForeignKeyStruct struct {
	Key              string // key in the table that references the foreign table
	ForeignTableName string
}

//--------------- KeyStruct END ------------------//
