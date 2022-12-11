package mongo

import (
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/models"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/utils"
	"go.mongodb.org/mongo-driver/bson"
)

// SortQueryBuilder creates sort object which support multiple column sort
func SortQueryBuilder(r *models.DataTableRequest, uniqueSortColumn string) bson.D {
	sortBson := bson.D{}
	for _, order := range r.Orders {
		intSortDirection := utils.ConvertOrderEnumToMongoInt(order.Dir)
		if intSortDirection != 0 {
			sortBson = append(sortBson, bson.E{
				Key:   r.Columns[order.Column].Data, // column name
				Value: intSortDirection,             // column direction as int value (0/1)
			})
		}
	}
	// Always add unique id column at the end to be able fetch non-unique columns in order
	sortBson = append(sortBson, bson.E{Key: uniqueSortColumn, Value: 1})
	return sortBson
}
