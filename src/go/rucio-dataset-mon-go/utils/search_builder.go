package utils

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"strconv"
	"strings"
)

// KB to EB bytes definition, uses x1024, not x1000
const (
	_          = iota // ignore first value by assigning to blank identifier
	KB float64 = 1 << (10 * iota)
	MB
	GB
	TB
	PB
	EB
)

// strToFloat converts user string to float64
func strToFloat(str string, typeAbbreviation string) float64 {
	// Remove space
	if strings.Contains(str, " ") {
		str = strings.ReplaceAll(str, " ", "")
	}
	str = strings.ReplaceAll(str, typeAbbreviation, "")
	f, err := strconv.ParseFloat(str, 10)
	if err != nil {
		ErrorLog("Cannot parse string to int: %s", str)
		return 0
	}
	InfoLogV1("strToFloat %f", f)
	return f
}

// humanSizeToBytes converts user defined size string to bytes
func humanSizeToBytes(input string) int64 {
	InfoLogV1("humanSizeToBytes input: %s", input)
	input = strings.ToUpper(input)
	switch {
	case strings.Contains(input, "KB"):
		return int64(strToFloat(input, "KB") * KB)
	case strings.Contains(input, "MB"):
		return int64(strToFloat(input, "MB") * MB)
	case strings.Contains(input, "GB"):
		return int64(strToFloat(input, "GB") * GB)
	case strings.Contains(input, "TB"):
		return int64(strToFloat(input, "TB") * TB)
	case strings.Contains(input, "PB"):
		return int64(strToFloat(input, "PB") * PB)
	case strings.Contains(input, "EB"):
		return int64(strToFloat(input, "EB") * EB)
	default:
		return int64(strToFloat(input, ""))
	}
}

// searchBsonSelections creates bson.M using SearchBuilder request
//   In main DataTables, there are 3 types in our data: string,num,date
//   - string:
//       It only has "contains" condition which is behaved as ReGex.
//       Since regex is powerful, we don't need starts with, ends with etc. conditions
//   - html:
//       [IMPORTANT] we use this type for numeric types columns. Because "num" type do not provide whole string like "10TB", only "10"
//       It has only "starts:Greater Than" and "ends:Less Than", in other words greater than and less than conditions.
//       They behaved as $lte and $gte
//       User may provide humanized size definition like "10 TB", it is converted to bytes to use in MongoDB queries
//   - date:
//       It has only "<", ">", "between", "null", "!null" conditions.
//       In other words: before, after, between, empty, not empty
//       TODO Currently it operates on string level, using LastAccessMs column(integer unix timestamp)
//           we can convert user date string to unix ts and use it in MongoDb query on LastAccessMs column
func searchBsonSelections(criterion models.SingleCriteria) bson.M {
	switch criterion.Type {
	case "string":
		switch criterion.Condition {
		case "contains": // String type should have only "contains" and regex applies
			return bson.M{criterion.OrigData: primitive.Regex{Pattern: criterion.Value[0], Options: "im"}}
		default:
			ErrorLog(" searchBsonSelections failed type is: %s", criterion.Type)
		}
	case "html":
		switch criterion.Condition {
		case "starts":
			bytesFilter := humanSizeToBytes(criterion.Value[0])
			if bytesFilter == 0 {
				return bson.M{} // If string value is not parsable, return null query
			}
			return bson.M{criterion.OrigData: bson.M{"$gte": bytesFilter}}
		case "ends":
			bytesFilter := humanSizeToBytes(criterion.Value[0])
			if bytesFilter == 0 {
				return bson.M{} // If string value is not parsable, return null query
			}
			log.Println(bytesFilter)
			return bson.M{criterion.OrigData: bson.M{"$lte": bytesFilter}}
		default:
			ErrorLog(" searchBsonSelections failed type is: %s", criterion.Type)
		}
	case "date":
		// TODO change the logic to use LastAccessMs
		switch criterion.Condition {
		case "<":
			return bson.M{criterion.OrigData: bson.M{"$lte": criterion.Value[0]}}
		case ">":
			return bson.M{criterion.OrigData: bson.M{"$gte": criterion.Value[0]}}
		case "between":
			return bson.M{
				"$and": []bson.M{
					{criterion.OrigData: bson.M{"$gte": criterion.Value[0]}},
					{criterion.OrigData: bson.M{"$lte": criterion.Value[1]}},
				}}
		case "null":
			return bson.M{criterion.OrigData: bson.M{"$exists": false}}
		case "!null":
			return bson.M{criterion.OrigData: bson.M{"$exists": true}}
		default:
			ErrorLog(" searchBsonSelections failed type is: %s", criterion.Type)
		}
	}
	return bson.M{}
}

// GetSearchBuilderBson iterates over all criteria(s) and creates "AND"/"OR" bson.M query
func GetSearchBuilderBson(sb *models.SearchBuilderRequest) bson.M {
	var andQuery []bson.M
	for _, condition := range sb.Criteria {
		andQuery = append(andQuery, searchBsonSelections(condition))
	}
	switch sb.Logic {
	case "AND":
		return bson.M{"$and": andQuery}
	case "OR":
		return bson.M{"$or": andQuery}
	default:
		return bson.M{}
	}
}
